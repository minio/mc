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
	"strings"
	"time"

	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go"

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
  {{end}}{{end}}
EXAMPLES:
  1. Upload 200MiB of a file to a bucket in 5 parts of size 40MiB.
      {{.HelpName}} if=file.txt of=play/my-bucket/file.txt size=40MiB parts=5

  2. Upload a full file to a bucket with 40MiB parts.
      {{.HelpName}} if=file.txt of=play/my-bucket/file.txt size=40MiB

  3. Upload a full file to a bucket in 5 parts.
      {{.HelpName}} if=file.txt of=play/my-bucket/file.txt parts=5
`,
}

type odPutMessage struct {
	Status    string `json:"status"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	PartSize  uint64 `json:"partSize"`
	TotalSize int64  `json:"totalSize"`
	Parts     int    `json:"parts"`
	Elapsed   int64  `json:"elapsed"`
}

type odGetMessage struct {
	Status    string `json:"status"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	TotalSize int64  `json:"totalSize"`
	Parts     int    `json:"parts"`
	Elapsed   int64  `json:"elapsed"`
}

func (o odPutMessage) String() string {
	cleanSize := humanize.IBytes(uint64(o.TotalSize))
	elapsed := time.Duration(o.Elapsed) * time.Millisecond
	return fmt.Sprintf("Transferred: %s, Parts: %d, Time: %s", cleanSize, o.Parts, elapsed)
}

func (o odPutMessage) JSON() string {
	copyMessageBytes, e := json.MarshalIndent(o, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(copyMessageBytes)
}

func (o odGetMessage) String() string {
	cleanSize := humanize.IBytes(uint64(o.TotalSize))
	elapsed := time.Duration(o.Elapsed) * time.Millisecond
	if o.Parts == 0 {
		return fmt.Sprintf("Transferred: %s, Full file, Time: %s", cleanSize, elapsed)
	}
	return fmt.Sprintf("Transferred: %s, Parts: %d, Time: %s", cleanSize, o.Parts, elapsed)
}

func (o odGetMessage) JSON() string {
	copyMessageBytes, e := json.MarshalIndent(o, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(copyMessageBytes)
}

// getOdUrls returns the URLs for the object download.
func getOdUrls(ctx context.Context, args madmin.KVS) (odURLs URLs, e error) {
	inFile := args.Get("if")
	outFile := args.Get("of")

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

	return odURLs, nil
}

// setSizesPut sets necessary values for object upload.
func setSizesPut(odURLs URLs, args madmin.KVS) (combinedSize int64, partSize uint64, parts int, e error) {
	// If parts not specified, set to 0, else scan for integer.
	p := args.Get("parts")
	if p == "" {
		parts = 0
	} else {
		parts, e = strconv.Atoi(p)
		if e != nil {
			return 0, 0, 0, e
		}
	}

	filesize := odURLs.SourceContent.Size

	s := args.Get("size")
	if parts <= 1 && s == "" {
		if parts == 0 {
			return 0, 0, 0, fmt.Errorf("either parts or size must be specified")
		}
		if parts == 1 {
			return filesize, uint64(filesize), 1, nil
		}
		return 0, 0, 0, fmt.Errorf("parts must be at least 1 or size must be specified")
	}

	// If size is not specified, calculate the size of each part and upload full file.
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

// setPartsGet sets parts for object download.
func setPartsGet(odURLs URLs, args madmin.KVS) (parts int, e error) {
	if args.Get("size") != "" {
		return 0, fmt.Errorf("size cannot be specified getting from server")
	}

	p := args.Get("parts")
	if p == "" {
		return 0, nil
	}
	parts, e = strconv.Atoi(p)
	if e != nil {
		return 0, e
	}
	if parts < 1 {
		return 0, fmt.Errorf("parts must be at least 1")
	}
	return parts, nil
}

// odPutOrGet checks if request is a download or upload and calls the appropriate function
func odPutOrGet(ctx context.Context, odURLs URLs, args madmin.KVS) (message, error) {
	if odURLs.SourceAlias != "" && odURLs.TargetAlias == "" {
		return odGet(ctx, odURLs, args)
	}
	if odURLs.SourceAlias == "" && odURLs.TargetAlias != "" {
		return odPut(ctx, odURLs, args)
	}
	return odPutMessage{}, fmt.Errorf("must download or upload, cannot copy locally or on server")
}

// odPut uploads the object.
func odPut(ctx context.Context, odURLs URLs, args madmin.KVS) (odPutMessage, error) {
	// Set sizes.
	combinedSize, partSize, parts, e := setSizesPut(odURLs, args)
	if e != nil {
		return odPutMessage{}, e
	}

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

	message := odPutMessage{
		Status:    "success",
		Source:    sourcePath,
		Target:    targetPath,
		PartSize:  partSize,
		TotalSize: total,
		Parts:     parts,
		Elapsed:   elapsed.Milliseconds(),
	}

	return message, nil
}

// odGet downloads the object.
func odGet(ctx context.Context, odURLs URLs, args madmin.KVS) (odGetMessage, error) {
	/// Set number of parts to get
	parts, e := setPartsGet(odURLs, args)
	if e != nil {
		return odGetMessage{}, e
	}

	targetPath := odURLs.TargetContent.URL.Path
	sourceAlias := odURLs.SourceAlias
	sourceURL := odURLs.SourceContent.URL
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))

	cli, err := newClientFromAlias(sourceAlias, sourceURL.String())
	fatalIf(err, "Unable to initialize client")

	var total int64
	var elapsed time.Duration
	if parts == 0 {
		// Get the file.
		total, elapsed = singleGet(ctx, cli, sourcePath, targetPath)
	} else {
		// Get the file in parts.
		total, elapsed = multiGet(ctx, cli, sourcePath, targetPath, parts)
	}

	message := odGetMessage{
		Status:    "success",
		Source:    sourcePath,
		Target:    targetPath,
		TotalSize: total,
		Parts:     parts,
		Elapsed:   elapsed.Milliseconds(),
	}

	return message, nil
}

// singleGet helps odGet download a single part.
func singleGet(ctx context.Context, cli Client, sourcePath, targetPath string) (total int64, elapsed time.Duration) {
	reader, err := cli.ODGet(ctx, 0)
	fatalIf(err, "Unable to get object reader")
	defer reader.Close()

	putOpts := PutOptions{
		disableMultipart: true,
	}

	pg := newAccounter(-1)

	// Upload the file.
	total, err = putTargetStream(ctx, "", targetPath, "", "", "",
		reader, -1, pg, putOpts)
	fatalIf(err, "Unable to download object")

	// Get upload time.
	elapsed = time.Since(pg.startTime)

	return total, elapsed
}

// multiGet helps odGet download multiple parts.
func multiGet(ctx context.Context, cli Client, sourcePath, targetPath string, parts int) (total int64, elapsed time.Duration) {
	var readers []io.Reader

	// Get reader for each part.
	for i := 1; i <= parts; i++ {
		reader, err := cli.ODGet(ctx, parts)
		fatalIf(err, "Unable to get object reader")
		readers = append(readers, reader)
		defer reader.Close()
	}
	reader := io.MultiReader(readers...)

	putOpts := PutOptions{
		disableMultipart: true,
	}

	// Unbounded Accounter to get time
	pg := newAccounter(-1)

	// Download the file.
	total, err := putTargetStream(ctx, "", targetPath, "", "", "",
		reader, -1, pg, putOpts)
	fatalIf(err, "Unable to download object")

	// Get upload time.
	elapsed = time.Since(pg.startTime)

	return total, elapsed
}

// mainOd is the entry point for the od command.
func mainOD(cliCtx *cli.Context) error {
	ctx, cancelCopy := context.WithCancel(globalContext)
	defer cancelCopy()

	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "od", 1) // last argument is exit code
	}

	var kvsArgs madmin.KVS
	for _, arg := range cliCtx.Args() {
		kv := strings.SplitN(arg, "=", 2)
		kvsArgs.Set(kv[0], kv[1])
	}

	// Get content from source.
	odURLs, e := getOdUrls(ctx, kvsArgs)
	fatalIf(probe.NewError(e), "Unable to get source and target URLs")

	message, e := odPutOrGet(ctx, odURLs, kvsArgs)
	fatalIf(probe.NewError(e), "Unable to transfer object")

	// Print message.
	printMsg(message)
	return nil
}
