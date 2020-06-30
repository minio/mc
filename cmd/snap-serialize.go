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

	"github.com/klauspost/compress/zstd"
	"github.com/minio/mc/pkg/probe"
	"github.com/tinylib/msgp/msgp"
)

// snapshotSerializer serializes snapshot data.
// Can be initialized with zero value data.
type snapshotSerializer struct {
	*msgp.Writer

	bucket string
	comp   *zstd.Encoder
	out    io.WriteCloser
}

// snapshotSerializeVersion identifies the version in case breaking changes
// must be made this can be bumped to detect
const snapshotSerializeVersion = 1

// A fancy 4 byte identifier and a 1 byte version.
var snapshotSerializeHeader = []byte{0x73, 0x4b, 0xe5, 0x6c, snapshotSerializeVersion}

// ResetFile will reset output to a file.
// Close should have been called before using this except if unused.
func (s *snapshotSerializer) ResetFile(snapName, bucket string) *probe.Error {
	w, perr := createSnapshotFile(snapName, filepath.Join("buckets", bucket))
	if perr != nil {
		return perr
	}
	s.out = w
	s.bucket = bucket

	// Add version.
	_, err := w.Write(snapshotSerializeHeader)
	if err != nil {
		return probe.NewError(err)
	}
	s.out = w
	if s.comp == nil {
		s.comp, err = zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithZeroFrames(true))
		if err != nil {
			return probe.NewError(err)
		}
	} else {
		s.comp.Reset(w)
	}

	if s.Writer == nil {
		s.Writer = msgp.NewWriter(s.comp)
	} else {
		s.Writer.Reset(s.comp)
	}
	return nil
}

// Close will write EOF marker and close the current output.
// If no output has been set nil is returned.
func (s *snapshotSerializer) Close() *probe.Error {
	if s.Writer == nil {
		return nil
	}
	// Write 255 as EOF.
	err := s.WriteInt8(255)
	if err != nil {
		return probe.NewError(err)
	}
	err = s.Writer.Flush()
	if err != nil {
		return probe.NewError(err)
	}
	err = s.comp.Flush()
	if err != nil {
		return probe.NewError(err)
	}
	err = s.out.Close()
	s.out = nil
	return probe.NewError(err)
}

// CleanUp will clean up the compressor.
func (s *snapshotSerializer) CleanUp() {
	if s.comp != nil {
		s.comp.Close()
	}
	// Be sure we close the file.
	if s.out != nil {
		s.out.Close()
		s.out = nil
	}
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
