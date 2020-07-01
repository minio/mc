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
	"time"

	"github.com/cespare/xxhash"
)

type SnapshotBucket struct {
	ID string `msg:"id"`
}

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

type S3Target struct {
	URL          string `msg:"url"`
	AccessKey    string `msg:"accessKey"`
	SecretKey    string `msg:"secretKey"`
	SessionToken string `msg:"sessionToken,omitempty"`
	API          string `msg:"api"`
	Lookup       string `msg:"lookup"`
}

// packetType is the type of a packet in the serialization format.
type packetType uint8

const (
	// Keep zero value unused for error detection.
	typeInvalid packetType = iota
	typeTargetStart
	typeBucketHeader
	typeBucketEntries
	typeTargetEnd
)

const (
	// Entries >= typeSkip are safe to skip if not known.
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

func (s *snapPacket) reset(t packetType) {
	s.Type = t
	s.Payload = s.Payload[:0]
	if cap(s.CRC) < 4 {
		s.CRC = make([]byte, 0, 4)
	}
	s.CRC = s.CRC[:0]
}

func (s *snapPacket) calcCRC() {
	h := xxhash.Sum64(s.Payload)
	s.CRC = s.CRC[:4]
	binary.LittleEndian.PutUint32(s.CRC, uint32(h)^uint32(h>>32))
}
