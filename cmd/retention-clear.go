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
	retentionClearFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "clear retention recursively",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "clear retention of a specific object version",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "roll back object(s) to current version at specified time",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "clear retention of object(s) and all its versions",
		},
		cli.BoolFlag{
			Name:  "default",
			Usage: "set default bucket locking",
		},
	}
)

var retentionClearCmd = cli.Command{
	Name:         "clear",
	Usage:        "clear retention for object(s)",
	Action:       mainRetentionClear,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(retentionClearFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Clear object retention for a specific object
     $ {{.HelpName}} myminio/mybucket/prefix/obj.csv

  2. Clear object retention for recursively for all objects at a given prefix
     $ {{.HelpName}} myminio/mybucket/prefix --recursive

  3. Clear object retention for a specific version of a specific object
     $ {{.HelpName}} myminio/mybucket/prefix/obj.csv --version-id "3Jr2x6fqlBUsVzbvPihBO3HgNpgZgAnp"

  4. Clear object retention for recursively for all versions of all objects
     $ {{.HelpName}} myminio/mybucket/prefix --recursive --versions

  5. Clear object retention for recursively for all versions created one year ago
     $ {{.HelpName}} myminio/mybucket/prefix --recursive --versions --rewind 365d

  6. Clear a bucket retention configuration
     $ {{.HelpName}} --default myminio/mybucket/
`,
}

func parseClearRetentionArgs(cliCtx *cli.Context) (target, versionID string, timeRef time.Time, withVersions, recursive, bucketMode bool) {
	args := cliCtx.Args()
	target = args[0]
	if target == "" {
		fatalIf(errInvalidArgument().Trace(), "invalid target url '%v'", target)
	}

	versionID = cliCtx.String("version-id")
	timeRef = parseRewindFlag(cliCtx.String("rewind"))
	withVersions = cliCtx.Bool("versions")
	recursive = cliCtx.Bool("recursive")
	bucketMode = cliCtx.Bool("default")
	return
}

// Clear Retention for one object/version or many objects within a given prefix, bypass governance is always enabled
func clearRetention(ctx context.Context, target, versionID string, timeRef time.Time, withOlderVersions, isRecursive bool) error {
	return applyRetention(ctx, lockOpClear, target, versionID, timeRef, withOlderVersions, isRecursive, "", 0, minio.Days, true)
}

func clearBucketLock(urlStr string) error {
	return applyBucketLock(lockOpClear, urlStr, "", 0, "")
}

// main for retention clear command.
func mainRetentionClear(cliCtx *cli.Context) error {
	ctx, cancelSetRetention := context.WithCancel(globalContext)
	defer cancelSetRetention()

	console.SetColor("RetentionSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("RetentionFailure", color.New(color.FgYellow))

	if len(cliCtx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(cliCtx, "clear", 1)
	}

	target, versionID, rewind, withVersions, recursive, bucketMode := parseClearRetentionArgs(cliCtx)

	checkObjectLockSupport(ctx, target)

	if bucketMode {
		return clearBucketLock(target)
	}

	if withVersions && rewind.IsZero() {
		rewind = time.Now().UTC()
	}

	return clearRetention(ctx, target, versionID, rewind, withVersions, recursive)
}
