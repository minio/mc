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

//go:generate msgp -unexported $GOFILE

package cmd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/cespare/xxhash"
	"github.com/minio/mc/pkg/probe"
)

// snapshotSerializeVersion identifies the version in case breaking changes
// must be made this can be bumped to detect.
const snapshotSerializeVersion = 1

// Serialized types can be extended mostly with the same rules as JSON:
//
// New fields have zero values.
// Removed fields are ignored.
//
// Breaking changes:
//
// Renaming the *serialized* name will break require conversion.
// Changing the type will in most cases be breaking.
// A breaking change can be implemented by bumping snapshotSerializeVersion and providing conversion.
//
// If a breaking change must be done the serializer keeps track of the saved version.
// That way older versions can be de-serialized and converted.
// If the saved snapshotSerializeVersion is bigger than the current

// SnapshotBucket is a msgp entry indicating this is a new bucket
type SnapshotBucket struct {
	Name string `msg:"name"`
}

// SnapshotEntry represents an S3 object
type SnapshotEntry struct {
	Key            string    `msg:"k"`
	VersionID      string    `msg:"vid"`
	Size           int64     `msg:"s"`
	ModTime        time.Time `msg:"mt"`
	StorageClass   string    `msg:"sc"`
	ETag           string    `msg:"etag"`
	IsDeleteMarker bool      `msg:"idm"`
	IsLatest       bool      `msg:"il"`
}

// S3Target represents the S3 endpoint
type S3Target struct {
	URL          string `msg:"url"`
	AccessKey    string `msg:"accessKey"`
	SecretKey    string `msg:"secretKey"`
	SessionToken string `msg:"sessionToken,omitempty"`
	API          string `msg:"api"`
	Lookup       string `msg:"lookup"`
}

// packetType is the type of a packet in the serialization format.
// New packet types should bump snapshotSerializeVersion unless the
// packets are 'skippable', meaning ID >= 127 in which case they are
// ignored by older versions.
// Changes in packet ordering are also breaking and require a version bump.
type packetType uint8

const (
	// Keep zero value unused for error detection.
	typeInvalid packetType = iota

	// Always first packet on stream.
	typeTargetStart

	// Always last packet on stream.
	typeTargetEnd

	// Indicates a bucket starting.
	// Each bucket should only be represented once on a stream.
	// Must end with typeBucketEnd and cannot be nested.
	typeBucketHeader

	// Can only be between typeBucketHeader and typeBucketEnd.
	// More than one packet is allowed consecutively.
	typeBucketEntries

	// Indicates end of a bucket.
	typeBucketEnd
)

const (
	// Entries >= typeSkip are safe to skip if not known.
	// typeSkip is reserved.
	typeSkip packetType = iota + 127
)

// Packet format for stream.
// Adding/removing fields will be a breaking change.
//msgp:tuple snapPacket
type snapPacket struct {
	Type    packetType
	CRC     []byte // optional, only if payload doesn't contain it.
	Payload []byte
}

// reset the packet and prepare for a new payload.
func (s *snapPacket) reset(t packetType) {
	s.Type = t
	s.Payload = s.Payload[:0]
	if cap(s.CRC) < 4 {
		s.CRC = make([]byte, 0, 4)
	}
	s.CRC = s.CRC[:0]
}

// calcCRC will calculate a CRC of the current payload.
func (s *snapPacket) calcCRC() {
	if len(s.Payload) == 0 {
		return
	}
	h := xxhash.Sum64(s.Payload)
	s.CRC = s.CRC[:4]
	binary.LittleEndian.PutUint32(s.CRC, uint32(h)^uint32(h>>32))
}

// Returns true if there is CRC and it matches.
func (s *snapPacket) CRCok() *probe.Error {
	if len(s.Payload) == 0 && len(s.CRC) == 0 {
		return nil
	}
	if len(s.CRC) != 4 {
		return probe.NewError(errors.New("want CRC value, but none was present"))
	}
	want := binary.LittleEndian.Uint32(s.CRC)
	h := xxhash.Sum64(s.Payload)
	got := uint32(h) ^ uint32(h>>32)
	if want != got {
		probe.NewError(fmt.Errorf("crc mismatch: want %x, got %x", want, got))
	}
	return nil
}

// skippable returns whether a packet type is skippable.
func (s *snapPacket) skippable() bool {
	return s.Type >= typeSkip
}
