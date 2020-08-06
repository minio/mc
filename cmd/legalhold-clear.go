/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
)

var (
	lhClearFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply legal hold recursively",
		},
		cli.StringFlag{
			Name:  "version-id",
			Usage: "apply legal hold to a specific object version",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "Move back in time",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "Pick earlier versions",
		},
	}
)
var legalHoldClearCmd = cli.Command{
	Name:   "clear",
	Usage:  "clear legal hold for object(s)",
	Action: mainLegalClearHold,
	Before: setGlobalsFromContext,
	Flags:  append(lhClearFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
   1. Disable legal hold on a specific object
      $ {{.HelpName}} myminio/mybucket/prefix/obj.csv

   2. Disable legal hold on a specific object version
      $ {{.HelpName}} myminio/mybucket/prefix/obj.csv --version-id "HiMFUTOowG6ylfNi4LKxD3ieHbgfgrvC"

   3. Disable object legal hold recursively for all objects at a prefix
      $ {{.HelpName}} myminio/mybucket/prefix --recursive

   4. Disable object legal hold recursively for all objects versions older than one year
      $ {{.HelpName}} myminio/mybucket/prefix --recursive --rewind 365d --versions
 `,
}

// main for retention command.
func mainLegalClearHold(ctx *cli.Context) error {
	console.SetColor("LegalHoldSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("LegalHoldPartialFailure", color.New(color.FgRed, color.Bold))
	console.SetColor("LegalHoldMessageFailure", color.New(color.FgYellow))

	args := ctx.Args()
	if len(args) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "clear", 1)
	}

	versionID := ctx.String("version-id")
	rewind := parseRewindFlag(ctx.String("rewind"))
	recursive := ctx.Bool("recursive")
	withVersions := ctx.Bool("versions")

	if versionID != "" && recursive {
		fatalIf(errInvalidArgument(), "Cannot pass --version-id and --recursive at the same time")
	}

	if rewind.IsZero() && withVersions {
		rewind = time.Now().UTC()
	}

	urlStr := args[0]
	return setLegalHold(urlStr, versionID, rewind, withVersions, recursive, minio.LegalHoldDisabled)
}
