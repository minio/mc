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
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

// put command flags.
var (
	putFlags = []cli.Flag{
		checksumFlag,
		cli.IntFlag{
			Name:  "parallel, P",
			Usage: "upload number of parts in parallel",
			Value: 4,
		},
		cli.StringFlag{
			Name:  "part-size, s",
			Usage: "each part size",
			Value: "16MiB",
		},
		cli.BoolFlag{
			Name:   "if-not-exists",
			Usage:  "upload only if object does not exist",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:  "disable-multipart",
			Usage: "disable multipart upload feature",
		},
		cli.StringFlag{
			Name:  "storage-class, sc",
			Usage: "set storage class for new object on target",
		},
	}
)

// Put command.
var putCmd = cli.Command{
	Name:         "put",
	Usage:        "upload an object to a bucket",
	Action:       mainPut,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(encFlags, globalFlags...), putFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

ENVIRONMENT VARIABLES:
  MC_ENC_KMS: KMS encryption key in the form of (alias/prefix=key).
  MC_ENC_S3: S3 encryption key in the form of (alias/prefix=key).

EXAMPLES:
  1. Put an object from local file system to S3 storage
     {{.Prompt}} {{.HelpName}} path-to/object play/mybucket

  2. Put an object from local file system to S3 bucket with name
     {{.Prompt}} {{.HelpName}} path-to/object play/mybucket/object

  3. Put an object from local file system to S3 bucket under a prefix
     {{.Prompt}} {{.HelpName}} path-to/object play/mybucket/object-prefix/

  4. Put an object to MinIO storage using sse-c encryption
     {{.Prompt}} {{.HelpName}} --enc-c "play/mybucket/object=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA" path-to/object play/mybucket/object 

  5. Put an object to MinIO storage using sse-kms encryption
     {{.Prompt}} {{.HelpName}} --enc-kms path-to/object play/mybucket/object

  6. Put an object to MinIO storage and assign REDUCED_REDUNDANCY storage-class to the uploaded object.
      {{.Prompt}} {{.HelpName}} --storage-class REDUCED_REDUNDANCY myobject.txt play/mybucket
`,
}

// mainPut is the entry point for put command.
func mainPut(cliCtx *cli.Context) (e error) {
	args := cliCtx.Args()
	if len(args) < 2 {
		showCommandHelpAndExit(cliCtx, 1) // last argument is exit code.
	}

	ctx, cancelPut := context.WithCancel(globalContext)
	defer cancelPut()

	// part size
	size := cliCtx.String("s")
	if size == "" {
		size = "32MiB"
	}

	_, perr := humanize.ParseBytes(size)
	if perr != nil {
		fatalIf(probe.NewError(perr), "Unable to parse part size")
	}
	// threads
	threads := cliCtx.Int("P")
	if threads < 1 {
		fatalIf(errInvalidArgument().Trace(strconv.Itoa(threads)), "Invalid number of threads")
	}

	disableMultipart := cliCtx.Bool("disable-multipart")

	// Parse encryption keys per command.
	encryptionKeys, err := validateAndCreateEncryptionKeys(cliCtx)
	if err != nil {
		err.Trace(cliCtx.Args()...)
	}
	fatalIf(err, "SSE Error")
	md5, checksum := parseChecksum(cliCtx)

	if len(args) < 2 {
		fatalIf(errInvalidArgument().Trace(args...), "Invalid number of arguments.")
	}

	// Check and handle storage class if passed in command line args
	storageClass := cliCtx.String("sc")

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
			encKeyDB:                encryptionKeys,
			ignoreBucketExistsCheck: true,
		}

		for putURLs := range preparePutURLs(ctx, opts) {
			if putURLs.Error != nil {
				putURLsCh <- putURLs
				break
			}
			if storageClass != "" {
				putURLs.TargetContent.StorageClass = storageClass
			}
			putURLs.checksum = checksum
			putURLs.MD5 = md5
			totalBytes += putURLs.SourceContent.Size
			pg.SetTotal(totalBytes)
			totalObjects++
			putURLs.DisableMultipart = disableMultipart
			putURLsCh <- putURLs
		}
		close(putURLsCh)
	}()
	for {
		select {
		case <-ctx.Done():
			showLastProgressBar(pg, nil)
			return
		case putURLs, ok := <-putURLsCh:
			if !ok {
				showLastProgressBar(pg, nil)
				return
			}
			if putURLs.Error != nil {
				printPutURLsError(&putURLs)
				showLastProgressBar(pg, putURLs.Error.ToGoError())
				return
			}
			urls := doCopy(ctx, doCopyOpts{
				cpURLs:           putURLs,
				pg:               pg,
				encryptionKeys:   encryptionKeys,
				multipartSize:    size,
				multipartThreads: strconv.Itoa(threads),
				ifNotExists:      cliCtx.Bool("if-not-exists"),
			})
			if urls.Error != nil {
				showLastProgressBar(pg, urls.Error.ToGoError())
				fatalIf(urls.Error.Trace(), "unable to upload")
				return
			}
		}
	}
}

func printPutURLsError(putURLs *URLs) {
	// Print in new line and adjust to top so that we
	// don't print over the ongoing scan bar
	if !globalQuiet && !globalJSON {
		console.Eraseline()
	}
	if strings.Contains(putURLs.Error.ToGoError().Error(),
		" is a folder.") {
		errorIf(putURLs.Error.Trace(),
			"Folder cannot be copied. Please use `...` suffix.")
	} else {
		errorIf(putURLs.Error.Trace(),
			"Unable to upload.")
	}
}

func showLastProgressBar(pg ProgressReader, e error) {
	if e != nil {
		// We only erase a line if we are displaying a progress bar
		if !globalQuiet && !globalJSON {
			console.Eraseline()
		}
		return
	}
	if progressReader, ok := pg.(*progressBar); ok {
		progressReader.Finish()
	} else {
		if accntReader, ok := pg.(*accounter); ok {
			printMsg(accntReader.Stat())
		}
	}
}
