/*
 * MinIO Client (C) 2019-2020 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
)

var (
	retentionSetFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply retention recursively",
		},
		cli.BoolFlag{
			Name:  "bypass",
			Usage: "bypass governance",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "apply retention to a specific object version",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "roll back object(s) to current version at specified time",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "apply retention object(s) and all its versions",
		},
		cli.BoolFlag{
			Name:  "default",
			Usage: "set bucket default retention mode",
		},
	}
)

var retentionSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set retention for object(s)",
	Action: mainRetentionSet,
	Before: setGlobalsFromContext,
	Flags:  append(retentionSetFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [governance | compliance] VALIDITY TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
VALIDITY:
  This argument must be formatted like Nd or Ny where 'd' denotes days and 'y' denotes years e.g. 10d, 3y.

EXAMPLES:
  1. Set object retention for a specific object
     $ {{.HelpName}} compliance 30d myminio/mybucket/prefix/obj.csv

  2. Set object retention for recursively for all objects at a given prefix
     $ {{.HelpName}} governance 30d myminio/mybucket/prefix --recursive

  3. Set object retention to a specific version of a specific object
     $ {{.HelpName}} governance 30d myminio/mybucket/prefix/obj.csv --version-id "3Jr2x6fqlBUsVzbvPihBO3HgNpgZgAnp"

  4. Set object retention for recursively for all versions of all objects
     $ {{.HelpName}} governance 30d myminio/mybucket/prefix --recursive --versions

  5. Set default lock retention configuration for a bucket
     $ {{.HelpName}} --default governance 30d myminio/mybucket/

`}

func parseSetRetentionArgs(cliCtx *cli.Context) (target, versionID string, recursive bool, timeRef time.Time, withVersions bool, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, bypass, bucketMode bool) {
	args := cliCtx.Args()
	mode = minio.RetentionMode(strings.ToUpper(args[0]))
	if !mode.IsValid() {
		fatalIf(errInvalidArgument().Trace(args...), "invalid retention mode '%v'", mode)
	}

	var err *probe.Error
	validity, unit, err = parseRetentionValidity(args[1])
	fatalIf(err.Trace(args[1]), "invalid validity argument")

	target = args[2]
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

// Set Retention for one object/version or many objects within a given prefix.
func setRetention(ctx context.Context, target, versionID string, timeRef time.Time, withOlderVersions, isRecursive bool,
	mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, bypassGovernance bool) error {
	return applyRetention(ctx, "set", target, versionID, timeRef, withOlderVersions, isRecursive, mode, validity, unit, bypassGovernance)
}

func setBucketLock(urlStr string, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit) error {
	return applyBucketLock("set", urlStr, mode, validity, unit)
}

// main for retention set command.
func mainRetentionSet(cliCtx *cli.Context) error {
	ctx, cancelSetRetention := context.WithCancel(globalContext)
	defer cancelSetRetention()

	console.SetColor("RetentionSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("RetentionFailure", color.New(color.FgYellow))

	if len(cliCtx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(cliCtx, "set", 1)
	}

	target, versionID, recursive, rewind, withVersions, mode, validity, unit, bypass, bucketMode := parseSetRetentionArgs(cliCtx)

	checkObjectLockSupport(ctx, target)

	if bucketMode {
		return setBucketLock(target, mode, validity, unit)
	}

	if withVersions && rewind.IsZero() {
		rewind = time.Now().UTC()
	}

	return setRetention(ctx, target, versionID, rewind, withVersions, recursive, mode, validity, unit, bypass)
}
