// Copyright (c) 2015-2021 MinIO, Inc.
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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
)

var (
	lhClearFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "clear legal hold recursively",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "clear legal hold of a specific object version",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "clear legal hold on an object version at specified time",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "clear legal hold on multiple versions of object(s)",
		},
	}
)

var legalHoldClearCmd = cli.Command{
	Name:         "clear",
	Usage:        "clear legal hold for object(s)",
	Action:       mainLegalHoldClear,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(lhClearFlags, globalFlags...),
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

// main for legalhold clear command.
func mainLegalHoldClear(cliCtx *cli.Context) error {
	console.SetColor("LegalHoldSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("LegalHoldPartialFailure", color.New(color.FgRed, color.Bold))
	console.SetColor("LegalHoldMessageFailure", color.New(color.FgYellow))

	targetURL, versionID, timeRef, recursive, withVersions := parseLegalHoldArgs(cliCtx)
	if timeRef.IsZero() && withVersions {
		timeRef = time.Now().UTC()
	}

	ctx, cancelCopy := context.WithCancel(globalContext)
	defer cancelCopy()

	enabled, err := isBucketLockEnabled(ctx, targetURL)
	if err != nil {
		fatalIf(err, "Unable to clear legalhold of `%s`", targetURL)
	}
	if !enabled {
		fatalIf(errDummy().Trace(), "Bucket locking needs to be enabled in order to use this feature.")
	}

	return setLegalHold(ctx, targetURL, versionID, timeRef, withVersions, recursive, minio.LegalHoldDisabled)
}
