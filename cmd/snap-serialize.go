/*
 * MinIO Client (C) 2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"
	"github.com/minio/mc/pkg/probe"
	"github.com/tinylib/msgp/msgp"
)

var zFastEnc *zstd.Encoder
var zFastEncInit sync.Once

func fastZstdEncoder() *zstd.Encoder {
	zFastEncInit.Do(func() {
		zFastEnc, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithWindowSize(1<<20))
	})
	return zFastEnc
}

// snapshotSerializer serializes snapshot data.
// Can be initialized with zero value data.
//
// Serialization format:
//
// All content is msgpack encoded.
//
// Header: Blob, must be 5 bytes, 4 bytes signature, 1 byte version.
//
// Packets: snapPacket objects
//  - Contains packet type (uint8) an optional CRC (4 byte blob)
//    and a blob payload depending on the packet type.
//
// Packets are order dependent.
type snapshotSerializer struct {
	msg *msgp.Writer
	mu  sync.Mutex

	packet   snapPacket
	out      io.Writer
	asyncErr error

	// Will be one if async error occurs
	hasAsyncErr int32
}

// snapshotSerializeVersion identifies the version in case breaking changes
// must be made this can be bumped to detect
const snapshotSerializeVersion = 1

// A fancy 4 byte identifier and a 1 byte version.
var snapshotSerializeHeader = []byte{0x73, 0x4b, 0xe5, 0x6c, snapshotSerializeVersion}

// ResetFile will reset output to a file.
// Close should have been called before using this except if unused.
func newSnapshotSerializer(w io.Writer) (*snapshotSerializer, *probe.Error) {
	var s snapshotSerializer
	s.out = w
	s.msg = msgp.NewWriter(w)

	// Add header+version.
	return &s, probe.NewError(s.msg.WriteBytes(snapshotSerializeHeader))
}

// AddTarget will add a target to the stream.
// The following belongs to the target until typeTargetEnd is on the stream.
func (s *snapshotSerializer) AddTarget(t S3Target) *probe.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	s.packet.reset(typeTargetStart)
	s.packet.Payload, err = t.MarshalMsg(s.packet.Payload[:0])
	if err != nil {
		return probe.NewError(err)
	}
	s.packet.calcCRC()
	return probe.NewError(s.packet.EncodeMsg(s.msg))
}

// StartBucket will start a bucket.
// Entries into the bucket can be written to the returned channel.
// The serializer should not be used until the channel has been closed.
func (s *snapshotSerializer) StartBucket(b SnapshotBucket) (chan<- SnapshotEntry, *probe.Error) {
	var err error
	s.packet.reset(typeBucketHeader)
	s.packet.Payload, err = b.MarshalMsg(s.packet.Payload[:0])
	if err != nil {
		return nil, probe.NewError(err)
	}
	s.packet.calcCRC()
	err = s.packet.EncodeMsg(s.msg)
	if err != nil {
		return nil, probe.NewError(err)
	}

	enc := fastZstdEncoder()
	encBlock := func(b []byte) {
		if len(b) == 0 || s.asyncErr != nil {
			return
		}
		s.packet.reset(typeBucketEntries)
		s.packet.Payload = enc.EncodeAll(b, s.packet.Payload[:0])
		// zstd has crc, so we don't add it again.
		s.setAsyncErr(s.packet.EncodeMsg(s.msg))
	}

	// Allow a reasonable buffer.
	entries := make(chan SnapshotEntry, 10000)
	go func() {
		const blockSize = 1 << 20
		// Make slightly larger temp block.
		tmp := make([]byte, 0, blockSize+1<<10)
		s.mu.Lock()
		defer s.mu.Unlock()

		for e := range entries {
			if s.asyncErr != nil {
				continue
			}
			tmp, err = e.MarshalMsg(tmp)
			s.setAsyncErr(err)
			if len(tmp) > blockSize {
				encBlock(tmp)
				tmp = tmp[:0]
			}
		}
		encBlock(tmp)
	}()
	return entries, nil
}

// hasError returns whether an async error has been recorded.
func (s *snapshotSerializer) HasError() bool {
	return atomic.LoadInt32(&s.hasAsyncErr) != 0
}

// getAsyncErr allows to get an async error, but requires that any operations are stopped.
func (s *snapshotSerializer) GetAsyncErr() error {
	s.mu.Lock()
	defer s.mu.Lock()
	return s.asyncErr
}

// setAsyncErr can be used to update the async error state.
// The caller must hold the serializer lock.
func (s *snapshotSerializer) setAsyncErr(err error) {
	if err == nil || s.asyncErr != nil {
		return
	}
	atomic.StoreInt32(&s.hasAsyncErr, 1)
	s.asyncErr = err
}

// FinishTarget will append a "snapshot end" packet and flush the output.
func (s *snapshotSerializer) FinishTarget() *probe.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.packet.reset(typeTargetEnd)
	err := s.packet.EncodeMsg(s.msg)
	if err != nil {
		probe.NewError(err)
	}
	return probe.NewError(s.msg.Flush())
}

// snapshotDeserializer handles de-serializing a bucket snapshot.
type snapshotDeserializer struct {
	*msgp.Reader

	bucket  string
	version uint8
	comp    *zstd.Decoder
	in      io.ReadCloser
}

// ResetFile will reset output to a file.
// Close should have been called before using this except if unused.
func newSnapShotReaderFile(snapName, bucket string) (*snapshotDeserializer, *probe.Error) {
	s := snapshotDeserializer{}
	r, perr := openSnapshotFile(filepath.Join(snapName, "buckets", bucket))
	if perr != nil {
		return nil, perr
	}
	s.in = r
	s.bucket = bucket

	// Read header + version.
	var tmp = make([]byte, len(snapshotSerializeHeader))
	_, err := io.ReadFull(r, tmp)
	if err != nil {
		return nil, probe.NewError(err)
	}
	if !bytes.Equal(tmp, snapshotSerializeHeader[:4]) {
		return nil, probe.NewError(errors.New("header signature mismatch"))
	}
	// Check version.
	switch tmp[4] {
	case 1:
	default:
		return nil, probe.NewError(errors.New("unknown content version"))
	}
	s.version = tmp[4]

	s.comp, err = zstd.NewReader(r, zstd.WithDecoderConcurrency(2))
	if err != nil {
		return nil, probe.NewError(err)
	}

	s.Reader = msgp.NewReader(s.comp)
	return &s, nil
}

// CleanUp will close file and clean up.
func (s *snapshotDeserializer) CleanUp() {
	s.in.Close()
	s.comp.Close()
}
