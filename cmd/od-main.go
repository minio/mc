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
  if=        source stream to upload
  of=        target path to upload to
  size=      size of each part. If not specified, will be calculated from the source stream size.
  parts=     number of parts to upload. If not specified, will calculated from the source file size.
  skip=      number of parts to skip.
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

type odMessage struct {
	Status    string `json:"status"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	PartSize  uint64 `json:"partSize"`
	TotalSize int64  `json:"totalSize"`
	Parts     int    `json:"parts"`
	Skip      int    `json:"skip"`
	Elapsed   int64  `json:"elapsed"`
}

func (o odMessage) String() string {
	cleanSize := humanize.IBytes(uint64(o.TotalSize))
	elapsed := time.Duration(o.Elapsed) * time.Millisecond
	speed := humanize.IBytes(uint64(float64(o.TotalSize) / elapsed.Seconds()))
	if o.Type == "S3toFS" && o.Parts == 0 {
		return fmt.Sprintf("Transferred: %s, Full file, Time: %s, Speed: %s/s", cleanSize, elapsed, speed)
	}
	return fmt.Sprintf("Transferred: %s, Parts: %d, Time: %s, Speed: %s/s", cleanSize, o.Parts, elapsed, speed)
}

func (o odMessage) JSON() string {
	odMessageBytes, e := json.MarshalIndent(o, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(odMessageBytes)
}

// getOdUrls returns the URLs for the object download.
func getOdUrls(ctx context.Context, args argKVS) (odURLs URLs, e error) {
	inFile := args.Get("if")
	outFile := args.Get("of")

	// Check if outFile is a folder or a file.
	opts := prepareCopyURLsOpts{
		sourceURLs: []string{inFile},
		targetURL:  outFile,
	}
	copyURLsContent, err := guessCopyURLType(ctx, opts)
	fatalIf(err, "Unable to guess copy URL type")

	// Get content of inFile, set up URLs.
	switch copyURLsContent.copyType {
	case copyURLsTypeA:
		odURLs = makeCopyContentTypeA(*copyURLsContent)
	case copyURLsTypeB:
		return URLs{}, fmt.Errorf("invalid source path %s, destination cannot be a directory", outFile)
	default:
		return URLs{}, fmt.Errorf("invalid source path %s, source cannot be a directory", inFile)
	}

	return odURLs, nil
}

// odCheckType checks if request is a download or upload and calls the appropriate function
func odCheckType(ctx context.Context, odURLs URLs, args argKVS) (message, error) {
	if odURLs.SourceAlias != "" && odURLs.TargetAlias == "" {
		return odDownload(ctx, odURLs, args)
	}

	var odType string
	if odURLs.SourceAlias == "" && odURLs.TargetAlias != "" {
		odType = "FStoS3"
	} else if odURLs.SourceAlias != "" && odURLs.TargetAlias != "" {
		odType = "S3toS3"
	} else {
		odType = "FStoFS"
	}
	return odCopy(ctx, odURLs, args, odType)
}

// mainOd is the entry point for the od command.
func mainOD(cliCtx *cli.Context) error {
	ctx, cancelCopy := context.WithCancel(globalContext)
	defer cancelCopy()

	if !cliCtx.Args().Present() {
		showCommandHelpAndExit(cliCtx, 1) // last argument is exit code
	}

	var kvsArgs argKVS
	for _, arg := range cliCtx.Args() {
		kv := strings.SplitN(arg, "=", 2)
		kvsArgs.Set(kv[0], kv[1])
	}

	// Get content from source.
	odURLs, e := getOdUrls(ctx, kvsArgs)
	fatalIf(probe.NewError(e), "Unable to get source and target URLs")

	message, e := odCheckType(ctx, odURLs, kvsArgs)
	fatalIf(probe.NewError(e), "Unable to transfer object")

	// Print message.
	printMsg(message)
	return nil
}
