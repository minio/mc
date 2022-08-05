// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"time"

	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"

	humanize "github.com/dustin/go-humanize"
)

type odUploadMessage struct {
	Status    string `json:"status"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	PartSize  uint64 `json:"partSize"`
	TotalSize int64  `json:"totalSize"`
	Parts     int    `json:"parts"`
	Elapsed   int64  `json:"elapsed"`
}

func (o odUploadMessage) String() string {
	cleanSize := humanize.IBytes(uint64(o.TotalSize))
	elapsed := time.Duration(o.Elapsed) * time.Millisecond
	return fmt.Sprintf("Transferred: %s, Parts: %d, Time: %s", cleanSize, o.Parts, elapsed)
}

func (o odUploadMessage) JSON() string {
	odMessageBytes, e := json.MarshalIndent(o, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(odMessageBytes)
}

// setSizesFStoS3 sets necessary values for object upload.
func setSizesFStoS3(odURLs URLs, args madmin.KVS) (combinedSize int64, partSize uint64, parts int, skip int64, e error) {
	// If parts not specified, set to 0, else scan for integer.
	p := args.Get("parts")
	if p == "" {
		parts = 0
	} else {
		parts, e = strconv.Atoi(p)
		if e != nil {
			return 0, 0, 0, 0, e
		}
	}

	filesize := odURLs.SourceContent.Size

	sk := args.Get("skip")
	var skipInt int
	if sk == "" {
		skipInt = 0
	} else {
		skipInt, e = strconv.Atoi(sk)
		if e != nil {
			return 0, 0, 0, 0, e
		}
	}

	s := args.Get("size")
	if parts <= 1 && s == "" {
		return filesize, uint64(filesize), 1, 0, nil
	}

	// If size is not specified, calculate the size of each part and upload full file.
	if s == "" {
		partSize = uint64(math.Ceil(float64(filesize) / float64(parts)))
		skip = int64(skipInt) * int64(partSize)
		parts = parts - skipInt
		return -1, partSize, parts, skip, nil
	}

	partSize, e = humanize.ParseBytes(s)
	if e != nil {
		return 0, 0, 0, 0, e
	}

	skip = int64(skipInt) * int64(partSize)

	combinedSize = int64(partSize * uint64(parts))
	if filesize == 0 {
		return combinedSize, partSize, parts, skip, nil
	}

	if partSize > uint64(filesize) {
		return -1, uint64(filesize), 1, 0, nil
	}

	// If parts is not specified, calculate number of parts and upload full file.
	if parts < 1 {
		combinedSize = filesize
		parts = int(math.Ceil(float64(combinedSize)/float64(partSize))) - skipInt
		return -1, partSize, parts, skip, nil
	}

	if filesize-skip < combinedSize {
		combinedSize = filesize - skip
		parts = int(math.Ceil(float64(filesize)/float64(partSize))) - skipInt
	}
	return combinedSize, partSize, parts, skip, nil
}

// odFStoS3 uploads the object.
func odFStoS3(ctx context.Context, odURLs URLs, args madmin.KVS) (odUploadMessage, error) {
	// Set sizes.
	combinedSize, partSize, parts, skip, e := setSizesFStoS3(odURLs, args)
	if e != nil {
		return odUploadMessage{}, e
	}

	sourcePath := odURLs.SourceContent.URL.Path
	targetAlias := odURLs.TargetAlias
	targetURL := odURLs.TargetContent.URL
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))

	getOpts := GetOptions{}
	if skip > 0 {
		getOpts.RangeStart = skip
	}

	var encKeyDB map[string][]prefixSSEPair

	// Create reader from source.
	reader, err := getSourceStreamFromURL(ctx, sourcePath, encKeyDB, getSourceOpts{GetOptions: getOpts})
	fatalIf(err, "Unable to get source stream")
	defer reader.Close()

	putOpts := PutOptions{
		storageClass:  odURLs.TargetContent.StorageClass,
		md5:           odURLs.MD5,
		multipartSize: partSize,
	}
	if parts == 1 {
		putOpts.disableMultipart = true
	}

	pg := newAccounter(combinedSize)

	// Upload the file.
	total, err := putTargetStream(ctx, targetAlias, targetURL.String(), "", "", "",
		reader, combinedSize, pg, putOpts)
	fatalIf(err, "Unable to upload file")

	// Get upload time.
	elapsed := time.Since(pg.startTime)

	message := odUploadMessage{
		Status:    "success",
		Type:      "FStoS3",
		Source:    sourcePath,
		Target:    targetPath,
		PartSize:  partSize,
		TotalSize: total,
		Parts:     parts,
		Elapsed:   elapsed.Milliseconds(),
	}

	return message, nil
}
