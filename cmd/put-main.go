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
	"time"

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
	Usage:        "upload local objects to object storage",
	Action:       mainPut,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(append(cpFlags, ioFlags...), globalFlags...), putFlags...),
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
      {{.Prompt}} {{.HelpName}} Music/*.ogg s3/jukebox/
`,
}

// mainPut is the entry point for cp command.
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

	// get source and target
	sourceURLs := cliCtx.Args()[:len(cliCtx.Args())-1]
	targetURL := cliCtx.Args()[len(cliCtx.Args())-1]
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	cpURLsCh := make(chan URLs, 10000)
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
			sourceURLs:         sourceURLs,
			targetURL:          targetURL,
			isRecursive:        false,
			encKeyDB:           encKeyDB,
			olderThan:          "",
			newerThan:          "",
			timeRef:            time.Time{},
			versionID:          "",
			isZip:              false,
			ignoreBucketExists: true,
		}

		for cpURLs := range prepareCopyURLs(ctx, opts) {
			if cpURLs.Error != nil {
				printCopyURLsError(&cpURLs)
				break
			}
			totalBytes += cpURLs.SourceContent.Size
			pg.SetTotal(totalBytes)
			totalObjects++
			cpURLsCh <- cpURLs
		}
		close(cpURLsCh)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case cpURLs, ok := <-cpURLsCh:
			if !ok {
				return nil
			}
			_ = doCopy(ctx, cpURLs, pg, encKeyDB, false, false, false)
		}
	}
}
