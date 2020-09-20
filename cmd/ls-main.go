/*
 * MinIO Client (C) 2014-2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/ioutils"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
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
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "list buckets and objects",
	Action: mainList,
	Before: setGlobalsFromContext,
	Flags:  append(lsFlags, globalFlags...),
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

`,
}

var rewindSupportedFormat = []string{
	"2006.01.02",
	"2006.01.02T15:04",
	"2006.01.02T15:04:05",
	time.RFC3339,
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
			if duration, e := ioutils.ParseDurationTime(rewind); e == nil {
				if duration < 0 {
					fatalIf(probe.NewError(errors.New("negative duration is not supported")),
						"Unable to parse --rewind argument")
				}
				timeRef = time.Now().Add(-duration)
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
func checkListSyntax(ctx context.Context, cliCtx *cli.Context) ([]string, bool, bool, time.Time, bool) {
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
	withOlderVersions := cliCtx.Bool("versions")

	timeRef := parseRewindFlag(cliCtx.String("rewind"))
	if timeRef.IsZero() && withOlderVersions {
		timeRef = time.Now().UTC()
	}

	return args, isRecursive, isIncomplete, timeRef, withOlderVersions
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

	// check 'ls' cliCtx arguments.
	args, isRecursive, isIncomplete, timeRef, withOlderVersions := checkListSyntax(ctx, cliCtx)

	var cErr error
	for _, targetURL := range args {
		clnt, err := newClient(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
		if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) {
			var st *ClientContent
			st, err = clnt.Stat(ctx, StatOptions{incomplete: isIncomplete})
			if st != nil && err == nil && st.Type.IsDir() {
				targetURL = targetURL + string(clnt.GetURL().Separator)
				clnt, err = newClient(targetURL)
				fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
			}
		}

		if e := doList(ctx, clnt, isRecursive, isIncomplete, timeRef, withOlderVersions); e != nil {
			cErr = e
		}
	}
	return cErr
}
