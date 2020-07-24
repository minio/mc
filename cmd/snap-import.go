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
	"fmt"
	"io"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	snapImportFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "overwrite",
			Usage: "Allow overwriting snapshots",
		},
		cli.StringFlag{
			Name:  "retarget",
			Usage: "Retarget imported snapshot to another server alias",
		},
	}
)

var snapImport = cli.Command{
	Name:   "import",
	Usage:  "Import a snapshot from stdin",
	Action: mainSnapImport,
	Before: setGlobalsFromContext,
	Flags:  append(snapImportFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} {{.Usage}}

USAGE:
  {{.HelpName}} MY-SNAPSHOT-NAME

EXAMPLES:
  1. Import a new snapshot from a .snap file
      {{.Prompt}} {{.HelpName}} my-snapshot-name </path/to/snapshot.snap

`,
}

// validate command-line args.
func parseSnapImportSyntax(ctx *cli.Context) (snapName string) {
	args := ctx.Args()
	if len(args) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "import", globalErrorExitStatus)
	}

	return cleanSnapName(args.Get(0))
}

func importSnapshot(ctx *cli.Context, input io.Reader, snapName string) *probe.Error {
	f, err := createSnapshotFile(snapName, ctx.Bool("overwrite"))
	if err != nil {
		return err
	}

	var target *S3Target
	if alias := ctx.String("retarget"); len(alias) > 0 {
		_, _, hostCfg, err := expandAlias(alias)
		if err != nil {
			return err
		}
		if hostCfg == nil {
			return probe.NewError(fmt.Errorf("unknown target %q", alias))
		}
		t := S3Target(*hostCfg)
		target = &t
	}

	err = copySnapshot(f, input, target)
	if err != nil {
		f.Close()
		return err
	}
	return probe.NewError(f.Close())
}

// main entry point for snapshot import
func mainSnapImport(ctx *cli.Context) error {
	// Validate command-line args.
	snapName := parseSnapExportSyntax(ctx)

	// Create a snapshot.
	err := importSnapshot(ctx, os.Stdin, snapName)
	fatalIf(err.Trace(), "Unable to import the specified snapshot")
	return nil
}
