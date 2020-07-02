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
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"
	"github.com/minio/mc/pkg/probe"
	"github.com/tinylib/msgp/msgp"
)

var zstdEnc *zstd.Encoder
var zstdEncInit sync.Once
var zstdDec *zstd.Decoder
var zstdDecInit sync.Once

func fastZstdEncoder() *zstd.Encoder {
	zstdEncInit.Do(func() {
		zstdEnc, _ = zstd.NewWriter(nil, zstd.WithWindowSize(1<<20))
	})
	return zstdEnc
}

func zstdDecoder() *zstd.Decoder {
	zstdDecInit.Do(func() {
		zstdDec, _ = zstd.NewReader(nil)
	})
	return zstdDec
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

// newSnapshotSerializer will serialize to a supplied writer.
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

const bucketEntriesBlockSize = 1 << 20

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
	encBlock := func(b []byte) []byte {
		if len(b) == 0 || s.asyncErr != nil {
			return b[:0]
		}
		s.packet.reset(typeBucketEntries)
		s.packet.Payload = enc.EncodeAll(b, s.packet.Payload[:0])

		// zstd has crc, so we don't add it again.
		s.setAsyncErr(s.packet.EncodeMsg(s.msg))
		return b[:0]
	}

	// Allow a reasonable buffer.
	entries := make(chan SnapshotEntry, 10000)
	go func() {
		// Make slightly larger temp block.
		tmp := make([]byte, 0, bucketEntriesBlockSize+1<<10)
		s.mu.Lock()
		defer s.mu.Unlock()

		for e := range entries {
			if s.asyncErr != nil {
				continue
			}
			tmp, err = e.MarshalMsg(tmp)
			s.setAsyncErr(err)
			if len(tmp) >= bucketEntriesBlockSize {
				tmp = encBlock(tmp)
			}
		}
		if s.asyncErr == nil {
			tmp = encBlock(tmp)
		}
		if s.asyncErr == nil {
			// Add end bucket packet.
			s.packet.reset(typeBucketEnd)
			s.setAsyncErr(s.packet.EncodeMsg(s.msg))
		}
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
	defer s.mu.Unlock()
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
	msg     *msgp.Reader
	version uint8
	packet  snapPacket

	in io.Reader
}

func newSnapShotReader(r io.Reader) (*snapshotDeserializer, *probe.Error) {
	s := snapshotDeserializer{
		msg: msgp.NewReader(r),
		in:  r,
	}

	// Read header + version.
	sz, err := s.msg.ReadBytesHeader()
	if err != nil {
		return nil, probe.NewError(err)
	}
	if int(sz) != len(snapshotSerializeHeader) {
		return nil, probe.NewError(errors.New("header signature length mismatch"))
	}
	var tmp = make([]byte, sz)
	_, err = s.msg.ReadFull(tmp)
	if err != nil {
		return nil, probe.NewError(err)
	}
	if !bytes.Equal(tmp[:4], snapshotSerializeHeader[:4]) {
		return nil, probe.NewError(errors.New("header signature mismatch"))
	}
	// Check version.
	switch tmp[4] {
	case 1:
	default:
		return nil, probe.NewError(errors.New("unknown content version"))
	}
	s.version = tmp[4]

	return &s, nil
}

// nextPacket will read the next packet.
// It will only skip if type is typeSkip
func (s *snapshotDeserializer) nextPacket() *probe.Error {
	for {
		s.packet.reset(typeInvalid)
		err := s.packet.DecodeMsg(s.msg)
		if err != nil {
			return probe.NewError(err)
		}
		switch s.packet.Type {
		case typeSkip:
			continue
		case typeInvalid:
			return probe.NewError(errors.New("invalid packet type"))
		}
		return nil
	}
}

// nextPacketType will read the next packet.
// Unless it is skippable it must be the type specified.
func (s *snapshotDeserializer) nextPacketType(want packetType) *probe.Error {
	for {
		err := s.nextPacket()
		if err != nil {
			return err
		}
		switch s.packet.Type {
		case want:
			return nil
		default:
			if !s.packet.skippable() {
				return probe.NewError(fmt.Errorf("unexpected packet type: want %d, got %d", want, s.packet.Type))
			}
		}
		return nil
	}
}

// nextPacketOneOf will read the next packet with one of the specified types.
// Unless it is skippable it must be one of the type specified.
func (s *snapshotDeserializer) nextPacketOneOf(want ...packetType) *probe.Error {
	var ok [256]bool
	for _, v := range want {
		ok[v] = true
	}
	for {
		err := s.nextPacket()
		if err != nil {
			return err
		}
		if ok[s.packet.Type] {
			return nil
		}
		if !s.packet.skippable() {
			return probe.NewError(fmt.Errorf("unexpected packet type: want one of %v, got %d", want, s.packet.Type))
		}
		return nil
	}
}

// skipUntil will skip until another packet type is found.
func (s *snapshotDeserializer) skipUntilNot(t packetType, skipSkippable bool) *probe.Error {
	for {
		err := s.nextPacket()
		if err != nil {
			return err
		}
		switch s.packet.Type {
		case t:
			continue
		default:
			if skipSkippable && s.packet.skippable() {
				continue
			}
			return nil
		}
	}
}

// ReadTarget will read a target from the stream.
// If the next packet is not a target an error is returned.
func (s *snapshotDeserializer) ReadTarget() (*S3Target, *probe.Error) {
	if err := s.nextPacketType(typeTargetStart); err != nil {
		return nil, err
	}
	if err := s.packet.CRCok(); err != nil {
		return nil, err
	}
	var dst S3Target
	_, err := dst.UnmarshalMsg(s.packet.Payload)
	return &dst, probe.NewError(err)
}

// ReadBucket will read a bucket header from the stream.
// If the next packet is not a bucket an error is returned.
// If there are no more buckets for the target nil, nil is returned.
func (s *snapshotDeserializer) ReadBucket() (*SnapshotBucket, *probe.Error) {
	if err := s.nextPacketOneOf(typeBucketHeader, typeTargetEnd); err != nil {
		return nil, err
	}
	if s.packet.Type == typeTargetEnd {
		return nil, nil
	}
	if err := s.packet.CRCok(); err != nil {
		return nil, err
	}
	var dst SnapshotBucket
	_, err := dst.UnmarshalMsg(s.packet.Payload)
	return &dst, probe.NewError(err)
}

// FindBucket will read buckets until the specified bucket is found.
// Will return nil, nil if the bucket cannot be found.
func (s *snapshotDeserializer) FindBucket(bucket string) (*SnapshotBucket, *probe.Error) {
	for {
		if err := s.nextPacketOneOf(typeBucketHeader, typeTargetEnd); err != nil {
			return nil, err
		}
		switch s.packet.Type {
		case typeTargetEnd:
			return nil, nil
		case typeBucketHeader:
			if err := s.packet.CRCok(); err != nil {
				return nil, err
			}
			var dst SnapshotBucket
			if _, err := dst.UnmarshalMsg(s.packet.Payload); err != nil {
				probe.NewError(err)
			}
			if dst.Name == bucket {
				return &dst, nil
			}
			// Skip entries...
			if err := s.skipUntilNot(typeBucketEntries, true); err != nil {
				return nil, err
			}
		}
		// We should only get the types above.
		return nil, probe.NewError(errors.New("internal error: unexpected packet type"))
	}
}

// SkipBucketEntries will skip bucket entries.
func (s *snapshotDeserializer) SkipBucketEntries() *probe.Error {
	for {
		if err := s.skipUntilNot(typeBucketEntries, true); err != nil {
			return err
		}
		if s.packet.Type == typeBucketEnd {
			return nil
		}
		if !s.packet.skippable() {
			return probe.NewError(fmt.Errorf("unexpected packet type %d", s.packet))
		}
	}
}

// BucketEntries will return all bucket entries.
// The channel will be closed when there are no more entries.
// If an error occurs it will be returned and the channel will be closed.
func (s *snapshotDeserializer) BucketEntries(ctx context.Context, entries chan<- SnapshotEntry) *probe.Error {
	defer close(entries)
	var dst SnapshotEntry
	var tmp = make([]byte, bucketEntriesBlockSize+(1<<10))
	dec := zstdDecoder()
	done := ctx.Done()
	for {
		select {
		case <-done:
			return probe.NewError(ctx.Err())
		default:
		}
		if err := s.nextPacketType(typeBucketEntries); err != nil {
			if s.packet.Type == typeBucketEnd {
				err = nil
			}
			return err
		}
		var err error
		tmp, err = dec.DecodeAll(s.packet.Payload, tmp[:0])
		if err != nil {
			return probe.NewError(err)
		}
		todo := tmp
		for len(todo) > 0 {
			todo, err = dst.UnmarshalMsg(todo)
			if err != nil {
				return probe.NewError(err)
			}
			entries <- dst
		}
	}
}
