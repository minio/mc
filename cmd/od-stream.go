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
	"math"
	"path/filepath"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"
)

// odSetSizes sets necessary values for object transfer.
func odSetSizes(odURLs URLs, args argKVS) (combinedSize int64, partSize uint64, parts int, skip int64, e error) {
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

	// Get number of parts to skip, defaults to 0.
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

	sourceSize := odURLs.SourceContent.Size

	// If neither parts nor size is specified, copy full file in 1 part.
	s := args.Get("size")
	if parts <= 1 && s == "" {
		return sourceSize, uint64(sourceSize), 1, 0, nil
	}

	// If size is not specified, calculate the size based on parts and upload full file.
	if s == "" {
		partSize = uint64(math.Ceil(float64(sourceSize) / float64(parts)))
		skip = int64(skipInt) * int64(partSize)
		parts = parts - skipInt
		return -1, partSize, parts, skip, nil
	}

	partSize, e = humanize.ParseBytes(s)
	if e != nil {
		return 0, 0, 0, 0, e
	}

	// Convert skipInt to bytes.
	skip = int64(skipInt) * int64(partSize)

	// If source has no content, calculate combined size and return.
	// This is needed for when source is /dev/zero.
	if sourceSize == 0 {
		if parts == 0 {
			parts = 1
		}
		combinedSize = int64(partSize * uint64(parts))
		return combinedSize, partSize, parts, skip, nil
	}

	// If source is smaller than part size, upload in 1 part.
	if partSize > uint64(sourceSize) {
		return sourceSize, uint64(sourceSize), 1, 0, nil
	}

	// If parts is not specified, calculate number of parts and upload full file.
	if parts < 1 {
		parts = int(math.Ceil(float64(sourceSize)/float64(partSize))) - skipInt
		return sourceSize, partSize, parts, skip, nil
	}

	combinedSize = int64(partSize * uint64(parts))

	// If combined size is larger than source after skip, recalculate number of parts.
	if sourceSize-skip < combinedSize {
		combinedSize = sourceSize - skip
		parts = int(math.Ceil(float64(sourceSize)/float64(partSize))) - skipInt
	}
	return combinedSize, partSize, parts, skip, nil
}

// odCopy copies a file/object from local to server, server to server, or local to local.
func odCopy(ctx context.Context, odURLs URLs, args argKVS, odType string) (odMessage, error) {
	// Set sizes.
	combinedSize, partSize, parts, skip, e := odSetSizes(odURLs, args)
	if e != nil {
		return odMessage{}, e
	}

	sourceAlias := odURLs.SourceAlias
	sourceURL := odURLs.SourceContent.URL
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
	targetAlias := odURLs.TargetAlias
	targetURL := odURLs.TargetContent.URL
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))

	getOpts := GetOptions{}

	// Skip given number of parts.
	if skip > 0 {
		getOpts.RangeStart = skip
	}

	// Placeholder encryption key database.
	var encKeyDB map[string][]prefixSSEPair

	// Create reader from source.
	reader, err := getSourceStreamFromURL(ctx, sourcePath, encKeyDB, getSourceOpts{GetOptions: getOpts})
	fatalIf(err.Trace(sourcePath), "Unable to get source stream")
	defer reader.Close()

	putOpts := PutOptions{
		storageClass:  odURLs.TargetContent.StorageClass,
		md5:           odURLs.MD5,
		multipartSize: partSize,
	}

	// Disable multipart on files too small for multipart upload.
	if combinedSize < 5242880 && combinedSize > 0 {
		putOpts.disableMultipart = true
	}

	// Used to get transfer time
	pg := newAccounter(combinedSize)

	// Write to target.
	targetClnt, err := newClientFromAlias(targetAlias, targetURL.String())
	fatalIf(err.Trace(targetURL.String()), "Unable to initialize target client")

	// Put object.
	total, err := targetClnt.PutPart(ctx, reader, combinedSize, pg, putOpts)
	fatalIf(err.Trace(targetURL.String()), "Unable to upload")

	// Get upload time.
	elapsed := time.Since(pg.startTime)

	message := odMessage{
		Status:    "success",
		Type:      odType,
		Source:    sourcePath,
		Target:    targetPath,
		PartSize:  partSize,
		TotalSize: total,
		Parts:     parts,
		Skip:      int(uint64(skip) / partSize),
		Elapsed:   elapsed.Milliseconds(),
	}

	return message, nil
}

// odSetParts sets parts for object download.
func odSetParts(args argKVS) (parts, skip int, e error) {
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

// odDownload copies an object from server to local.
func odDownload(ctx context.Context, odURLs URLs, args argKVS) (odMessage, error) {
	/// Set number of parts to get.
	parts, skip, e := odSetParts(args)
	if e != nil {
		return odMessage{}, e
	}

	targetPath := odURLs.TargetContent.URL.Path
	sourceAlias := odURLs.SourceAlias
	sourceURL := odURLs.SourceContent.URL
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))

	// Get server client.
	cli, err := newClientFromAlias(sourceAlias, sourceURL.String())
	fatalIf(err, "Unable to initialize client")

	var reader io.Reader
	if parts == 0 {
		// Get the full file.
		reader = singleGet(ctx, cli)
	} else {
		// Get the file in parts.
		reader = multiGet(ctx, cli, parts, skip)
	}

	// Accounter to get transfer time.
	pg := newAccounter(-1)

	// Upload the file.
	total, err := putTargetStream(ctx, "", targetPath, "", "", "",
		reader, -1, pg, PutOptions{})
	fatalIf(err.Trace(targetPath), "Unable to upload an object")

	// Get upload time.
	elapsed := time.Since(pg.startTime)

	message := odMessage{
		Status:    "success",
		Type:      "S3toFS",
		Source:    sourcePath,
		Target:    targetPath,
		TotalSize: total,
		Parts:     parts,
		Skip:      skip,
		Elapsed:   elapsed.Milliseconds(),
	}

	return message, nil
}

// singleGet helps odDownload download a single part.
func singleGet(ctx context.Context, cli Client) io.ReadCloser {
	reader, err := cli.GetPart(ctx, 0)
	fatalIf(err, "Unable to download object")

	return reader
}

// multiGet helps odDownload download multiple parts.
func multiGet(ctx context.Context, cli Client, parts, skip int) io.Reader {
	var readers []io.Reader

	// Get reader for each part.
	for i := 1 + skip; i <= parts; i++ {
		reader, err := cli.GetPart(ctx, parts)
		fatalIf(err, "Unable to download part of an object")
		readers = append(readers, reader)
	}
	reader := io.MultiReader(readers...)

	return reader
}
