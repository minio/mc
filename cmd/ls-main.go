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
	"errors"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

// ls specific flags.
var (
	lsFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "rewind",
			Usage: "list all object versions no later than specified date",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "list all versions",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "list recursively",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "list incomplete uploads",
		},
		cli.BoolFlag{
			Name:  "summarize",
			Usage: "display summary information (number of objects, total size)",
		},
		cli.StringFlag{
			Name:  "storage-class, sc",
			Usage: "filter to specified storage class",
		},
		cli.BoolFlag{
			Name:  "zip",
			Usage: "list files inside zip archive (MinIO servers only)",
		},
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:         "ls",
	Usage:        "list buckets and objects",
	Action:       mainList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(lsFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List buckets on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} s3

  2. List buckets and all its contents from Amazon S3 cloud storage recursively.
     {{.Prompt}} {{.HelpName}} --recursive s3

  3. List all contents of mybucket on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} s3/mybucket/

  4. List all contents of mybucket on Amazon S3 cloud storage on Microsoft Windows.
     {{.Prompt}} {{.HelpName}} s3\mybucket\

  5. List files recursively on a local filesystem on Microsoft Windows.
     {{.Prompt}} {{.HelpName}} --recursive C:\Users\Worf\

  6. List incomplete (previously failed) uploads of objects on Amazon S3.
     {{.Prompt}} {{.HelpName}} --incomplete s3/mybucket

  7. List contents at a specific time in the past if the bucket versioning is enabled.
     {{.Prompt}} {{.HelpName}} --rewind 2020.01.01 s3/mybucket
     {{.Prompt}} {{.HelpName}} --rewind 2020.01.01T11:30 s3/mybucket
     {{.Prompt}} {{.HelpName}} --rewind 7d s3/mybucket

  8. List all contents versions if the bucket versioning is enabled.
     {{.Prompt}} {{.HelpName}} --versions s3/mybucket

  9. List all objects on mybucket, summarize the number of objects and total size.
     {{.Prompt}} {{.HelpName}} --summarize s3/mybucket/
  
  10. List all objects on mybucket, for the GLACIER storage class
     {{.Prompt}} {{.HelpName}} --storage-class 'GLACIER' s3/mybucket 
`,
}

var rewindSupportedFormat = []string{
	"2006.01.02",
	"2006.01.02T15:04",
	"2006.01.02T15:04:05",
	time.RFC3339,
	printDate,
}

// Parse rewind flag while considering the system local time zone
func parseRewindFlag(rewind string) (timeRef time.Time) {
	if rewind != "" {
		location, e := time.LoadLocation("Local")
		if e != nil {
			return
		}

		for _, format := range rewindSupportedFormat {
			if t, e := time.ParseInLocation(format, rewind, location); e == nil {
				timeRef = t
				break
			}
		}

		if timeRef.IsZero() {
			// rewind is not parsed, check if it is a duration instead
			if duration, e := ParseDuration(rewind); e == nil {
				if duration < 0 {
					fatalIf(probe.NewError(errors.New("negative duration is not supported")),
						"Unable to parse --rewind argument")
				}
				timeRef = time.Now().Add(-time.Duration(duration))
			}
		}

		if timeRef.IsZero() {
			// rewind argument still not parsed, error out
			fatalIf(probe.NewError(errors.New("unknown format")), "Unable to parse --rewind argument")
		}
	}
	return
}

// checkListSyntax - validate all the passed arguments
func checkListSyntax(cliCtx *cli.Context) ([]string, doListOptions) {
	args := cliCtx.Args()
	if !cliCtx.Args().Present() {
		args = []string{"."}
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}

	isRecursive := cliCtx.Bool("recursive")
	isIncomplete := cliCtx.Bool("incomplete")
	withVersions := cliCtx.Bool("versions")
	isSummary := cliCtx.Bool("summarize")
	listZip := cliCtx.Bool("zip")

	timeRef := parseRewindFlag(cliCtx.String("rewind"))

	if listZip && (withVersions || !timeRef.IsZero()) {
		fatalIf(errInvalidArgument().Trace(args...), "Zip file listing can only be performed on the latest version")
	}
	storageClasss := cliCtx.String("storage-class")
	opts := doListOptions{
		timeRef:      timeRef,
		isRecursive:  isRecursive,
		isIncomplete: isIncomplete,
		isSummary:    isSummary,
		withVersions: withVersions,
		listZip:      listZip,
		filter:       storageClasss,
	}
	return args, opts
}

// mainList - is a handler for mc ls command
func mainList(cliCtx *cli.Context) error {
	ctx, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	// Additional command specific theme customization.
	console.SetColor("File", color.New(color.Bold))
	console.SetColor("DEL", color.New(color.FgRed))
	console.SetColor("PUT", color.New(color.FgGreen))
	console.SetColor("VersionID", color.New(color.FgHiBlue))
	console.SetColor("VersionOrd", color.New(color.FgHiMagenta))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("Time", color.New(color.FgGreen))
	console.SetColor("Summarize", color.New(color.Bold))
	console.SetColor("SC", color.New(color.FgBlue))

	// check 'ls' cliCtx arguments.
	args, opts := checkListSyntax(cliCtx)

	var cErr error
	for _, targetURL := range args {
		clnt, err := newClient(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
		if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) {
			var st *ClientContent
			st, err = clnt.Stat(ctx, StatOptions{incomplete: opts.isIncomplete, includeVersions: opts.withVersions})
			if st != nil && err == nil && st.Type.IsDir() {
				targetURL = targetURL + string(clnt.GetURL().Separator)
				clnt, err = newClient(targetURL)
				fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
			}
		}
		if e := doList(ctx, clnt, opts); e != nil {
			cErr = e
		}
	}
	return cErr
}
