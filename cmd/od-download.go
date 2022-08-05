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
	"io"
	"path/filepath"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"
	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

type odDownloadMessage struct {
	Status    string `json:"status"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	TotalSize int64  `json:"totalSize"`
	Parts     int    `json:"parts"`
	Elapsed   int64  `json:"elapsed"`
}

func (o odDownloadMessage) String() string {
	cleanSize := humanize.IBytes(uint64(o.TotalSize))
	elapsed := time.Duration(o.Elapsed) * time.Millisecond
	if o.Parts == 0 {
		return fmt.Sprintf("Transferred: %s, Full file, Time: %s", cleanSize, elapsed)
	}
	return fmt.Sprintf("Transferred: %s, Parts: %d, Time: %s", cleanSize, o.Parts, elapsed)
}

func (o odDownloadMessage) JSON() string {
	odMessageBytes, e := json.MarshalIndent(o, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(odMessageBytes)
}

// setPartsS3toFS sets parts for object download.
func setPartsS3toFS(odURLs URLs, args madmin.KVS) (parts int, skip int, e error) {
	if args.Get("size") != "" {
		return 0, 0, fmt.Errorf("size cannot be specified getting from server")
	}

	p := args.Get("parts")
	if p == "" {
		return 0, 0, nil
	}
	parts, e = strconv.Atoi(p)
	if e != nil {
		return 0, 0, e
	}
	if parts < 1 {
		return 0, 0, fmt.Errorf("parts must be at least 1")
	}

	sk := args.Get("skip")
	if sk == "" {
		skip = 0
	} else {
		skip, e = strconv.Atoi(sk)
	}
	if e != nil {
		return 0, 0, e
	}

	return parts, skip, nil
}

// odS3toFS downloads the object.
func odS3toFS(ctx context.Context, odURLs URLs, args madmin.KVS) (odDownloadMessage, error) {
	/// Set number of parts to get
	parts, skip, e := setPartsS3toFS(odURLs, args)
	if e != nil {
		return odDownloadMessage{}, e
	}

	targetPath := odURLs.TargetContent.URL.Path
	sourceAlias := odURLs.SourceAlias
	sourceURL := odURLs.SourceContent.URL
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))

	cli, err := newClientFromAlias(sourceAlias, sourceURL.String())
	fatalIf(err, "Unable to initialize client")

	var reader io.Reader
	if parts == 0 {
		// Get the file.
		reader = singleGet(ctx, cli)
	} else {
		// Get the file in parts.
		reader = multiGet(ctx, cli, parts, skip)
	}

	putOpts := PutOptions{
		disableMultipart: true,
	}

	pg := newAccounter(-1)

	// Upload the file.
	total, err := putTargetStream(ctx, "", targetPath, "", "", "",
		reader, -1, pg, putOpts)
	fatalIf(err, "Unable to download object")

	// Get upload time.
	elapsed := time.Since(pg.startTime)

	message := odDownloadMessage{
		Status:    "success",
		Type:      "S3toFS",
		Source:    sourcePath,
		Target:    targetPath,
		TotalSize: total,
		Parts:     parts,
		Elapsed:   elapsed.Milliseconds(),
	}

	return message, nil
}

// singleGet helps odS3toFS download a single part.
func singleGet(ctx context.Context, cli Client) io.ReadCloser {
	reader, err := cli.ODGet(ctx, 0)
	fatalIf(err, "Unable to get object reader")

	return reader
}

// multiGet helps odS3toFS download multiple parts.
func multiGet(ctx context.Context, cli Client, parts, skip int) io.Reader {
	var readers []io.Reader

	// Get reader for each part.
	for i := 1 + skip; i <= parts; i++ {
		reader, err := cli.ODGet(ctx, parts)
		fatalIf(err, "Unable to get object reader")
		readers = append(readers, reader)
	}
	reader := io.MultiReader(readers...)

	return reader
}
