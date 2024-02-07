// Copyright (c) 2015-2024 MinIO, Inc.
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
	"os"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

// put command flags.
var (
	putFlags = []cli.Flag{
		cli.IntFlag{
			Name:  "parallel, P",
			Usage: "upload number of parts in parallel",
		},
		cli.StringFlag{
			Name:  "part-size, s",
			Usage: "each part size",
		},
	}
)

// Put command.
var putCmd = cli.Command{
	Name:         "put",
	Usage:        "upload local object to s3 object storage",
	Action:       mainPut,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(ioFlags, globalFlags...), putFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
  MC_ENCRYPT:      list of comma delimited prefixes
  MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

EXAMPLES:
  01. Put a list of objects from local file system to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} objectsName s3/jukebox/
`,
}

// mainPut is the entry point for put command.
func mainPut(cliCtx *cli.Context) error {
	ctx, cancelPut := context.WithCancel(globalContext)
	defer cancelPut()
	// part size
	size := cliCtx.String("s")
	if size == "" {
		size = "16mb"
	}
	_, perr := humanize.ParseBytes(size)
	if perr != nil {
		fatalIf(probe.NewError(perr), "Unable to parse part size")
	}
	os.Setenv("MC_UPLOAD_MULTIPART_SIZE", size)
	// threads
	threads := cliCtx.Int("P")
	if threads == 0 {
		threads = 1
	}
	os.Setenv("MC_UPLOAD_MULTIPART_THREADS", strconv.Itoa(threads))

	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	args := cliCtx.Args()
	if len(args) < 2 {
		fatalIf(errInvalidArgument().Trace(args...), "Invalid number of arguments.")
	}
	// get source and target
	sourceURLs := args[:len(args)-1]
	targetURL := args[len(args)-1]

	putURLsCh := make(chan URLs, 10000)
	var totalObjects, totalBytes int64

	// Store a progress bar or an accounter
	var pg ProgressReader

	// Enable progress bar reader only during default mode.
	if !globalQuiet && !globalJSON { // set up progress bar
		pg = newProgressBar(totalBytes)
	} else {
		pg = newAccounter(totalBytes)
	}
	go func() {
		opts := prepareCopyURLsOpts{
			sourceURLs:              sourceURLs,
			targetURL:               targetURL,
			encKeyDB:                encKeyDB,
			ignoreBucketExistsCheck: true,
		}

		for putURLs := range preparePutURLs(ctx, opts) {
			if putURLs.Error != nil {
				printCopyURLsError(&putURLs)
				break
			}
			totalBytes += putURLs.SourceContent.Size
			pg.SetTotal(totalBytes)
			totalObjects++
			putURLsCh <- putURLs
		}
		close(putURLsCh)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case putURLs, ok := <-putURLsCh:
			if !ok {
				return nil
			}
			urls := doCopy(ctx, putURLs, pg, encKeyDB, false, false, false)
			if urls.Error != nil {
				return urls.Error.ToGoError()
			}
		}
	}
}
