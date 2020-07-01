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
	"os"
	"path/filepath"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	snapExportFlags = []cli.Flag{}
)

// FIXME:
var snapExport = cli.Command{
	Name:   "export",
	Usage:  "Export a snapshot to JSON format",
	Action: mainSnapExport,
	Before: setGlobalsFromContext,
	Flags:  append(snapExportFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} COMMAND - {{.Usage}}

USAGE:
  {{.HelpName}} COMMAND

COMMAND:

EXAMPLES:
`,
}

// validate command-line args.
func parseSnapExportSyntax(ctx *cli.Context) (snapName string) {
	args := ctx.Args()
	if len(args) != 1 {
		// fatalIf(errors.New("wrong arguments"), "")
	}

	return args.Get(0)
}

func exportSnapshot(snapName string) *probe.Error {
	snapsDir, err := getSnapsDir()
	if err != nil {
		return err
	}

	snapDir := filepath.Join(snapsDir, snapName)
	if _, err := os.Stat(snapDir); err != nil {
		return probe.NewError(err)
	}

	e := compress(snapDir, os.Stdout)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// main entry point for snapshot create.
func mainSnapExport(ctx *cli.Context) error {

	// Validate command-line args.
	snapName := parseSnapExportSyntax(ctx)

	// Create a snapshot.
	fatalIf(exportSnapshot(snapName).Trace(), "Unable to export the specified snapshot")
	return nil
}
