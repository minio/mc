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
	"strings"
	"time"

	json "github.com/minio/colorjson"

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

// make a bucket.
var odCmd = cli.Command{
	Name:         "od",
	Usage:        "measure single stream upload and download",
	Action:       mainOD,
	Before:       setGlobalsFromContext,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [OPERANDS]

OPERANDS:
  if=        Source stream to upload
  of=        Target path to upload to
  size=      Size of each part. If not specified, will be calculated from the source stream size.
  parts=     Number of parts to upload. If not specified, will calculated from the source file size.

{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Upload 200MiB of a file to a bucket in 5 parts of size 40MiB.
      {{.HelpName}} if=file.txt of=play/my-bucket/file.txt size=40MiB parts=5

  2. Upload a full file to a bucket with 40MiB parts.
      {{.HelpName}} if=file.txt of=play/my-bucket/file.txt size=40MiB

  3. Upload a full file to a bucket in 5 parts.
      {{.HelpName}} if=file.txt of=play/my-bucket/file.txt parts=5

`,
}

type odMessage struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	PartSize  uint64 `json:"partSize"`
	TotalSize int64  `json:"totalSize"`
	Parts     int    `json:"parts"`
	Elapsed   string `json:"elapsed"`
}

func (o odMessage) String() string {
	cleanSize := humanize.IBytes(uint64(o.TotalSize))
	return fmt.Sprintf("`%s` -> `%s`\n Transferred: %s, Parts: %d, Time: %s",
		o.Source, o.Target, cleanSize, o.Parts, o.Elapsed)
}

func (o odMessage) JSON() string {
	copyMessageBytes, e := json.MarshalIndent(o, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(copyMessageBytes)
}

func setOdSizes(args map[string]string, odURLs URLs) (combinedSize int64, partSize uint64, parts int, e error) {
	// If parts not specified, set to 0, else scan for integer.
	p := args["parts"]
	if p == "" {
		parts = 0
	} else {
		parts, e = strconv.Atoi(p)
		if e != nil {
			return 0, 0, 0, e
		}
	}

	s := args["size"]
	if parts < 1 && s == "" {
		if parts == 0 {
			return 0, 0, 0, fmt.Errorf("either parts or size must be specified")
		}
		return 0, 0, 0, fmt.Errorf("parts must be at least 1 or size must be specified")
	}

	// If size is not specified, calculate the size of each part and upload full file.
	filesize := odURLs.SourceContent.Size
	if s == "" {
		combinedSize = filesize
		partSize = uint64(math.Ceil(float64(combinedSize) / float64(parts)))
		return combinedSize, partSize, parts, nil
	}

	partSize, e = humanize.ParseBytes(s)
	if e != nil {
		return 0, 0, 0, e
	}

	// If parts is not specified, calculate number of parts and upload full file.
	if parts < 1 {
		combinedSize = filesize
		parts = int(math.Ceil(float64(combinedSize) / float64(partSize)))
		return combinedSize, partSize, parts, nil
	}

	combinedSize = int64(partSize * uint64(parts))
	if filesize < combinedSize {
		return 0, 0, 0, fmt.Errorf("size of source file is smaller than the combined size of parts")
	}
	return combinedSize, partSize, parts, nil
}

func getOdUrls(args map[string]string, ctx context.Context) (odURLs URLs, e error) {
	inFile := args["if"]
	outFile := args["of"]

	// Placeholder encryption key database
	var encKeyDB map[string][]prefixSSEPair

	// Check if outFile is a folder or a file.
	opts := prepareCopyURLsOpts{
		sourceURLs: []string{inFile},
		targetURL:  outFile,
	}
	odType, _, err := guessCopyURLType(ctx, opts)
	fatalIf(err.Trace(), "Unable to guess copy URL type")

	// Get content of inFile, set up URLs.
	switch odType {
	case copyURLsTypeA:
		odURLs = prepareCopyURLsTypeA(ctx, inFile, "", outFile, encKeyDB)
	case copyURLsTypeB:
		odURLs = prepareCopyURLsTypeB(ctx, inFile, "", outFile, encKeyDB)
	default:
		return URLs{}, fmt.Errorf("invalid source path %s, source cannot be a directory", inFile)
	}

	// Check if source exists.
	if odURLs.SourceContent == nil {
		return URLs{}, fmt.Errorf("invalid source path %s", inFile)
	}

	// Check if target alias is valid.
	if odURLs.TargetAlias == "" {
		return URLs{}, fmt.Errorf("invalid target path %s", outFile)
	}

	return odURLs, nil
}

func mainOD(cliCtx *cli.Context) error {
	ctx, cancelCopy := context.WithCancel(globalContext)
	defer cancelCopy()

	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "mb", 1) // last argument is exit code
	}

	mapArgs, err := parseKVArgs(strings.Join(cliCtx.Args(), ","))
	fatalIf(err.Trace(), "Unable to parse arguments.")

	// Get content from source.
	odURLs, e := getOdUrls(mapArgs, ctx)
	fatalIf(probe.NewError(e), "Unable to get source and target URLs")

	// Set sizes.
	combinedSize, partSize, parts, e := setOdSizes(mapArgs, odURLs)
	fatalIf(probe.NewError(e), "Unable to set parts size")

	sourcePath := odURLs.SourceContent.URL.Path
	targetAlias := odURLs.TargetAlias
	targetURL := odURLs.TargetContent.URL
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))

	// Create reader from source.
	reader, metadata, err := getSourceStream(ctx, "", sourcePath, getSourceOpts{})
	fatalIf(err, "Unable to get source stream")
	defer reader.Close()

	putOpts := PutOptions{
		metadata:      filterMetadata(metadata),
		storageClass:  odURLs.TargetContent.StorageClass,
		md5:           odURLs.MD5,
		multipartSize: partSize,
	}
	pg := newAccounter(combinedSize)

	// Upload the file.
	_, err = putTargetStream(ctx, targetAlias, targetURL.String(), "", "", "",
		reader, combinedSize, pg, putOpts)
	fatalIf(err, "Unable to put target stream")

	// Get upload time.
	elapsed := time.Since(pg.startTime)

	printMsg(odMessage{
		Source:    sourcePath,
		Target:    targetPath,
		PartSize:  partSize,
		TotalSize: combinedSize,
		Parts:     parts,
		Elapsed:   elapsed.Round(time.Millisecond).String(),
	})
	return nil
}
