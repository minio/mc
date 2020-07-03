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
	"io"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	snapExportFlags = []cli.Flag{}
)

var snapExport = cli.Command{
	Name:   "export",
	Usage:  "Export a snapshot to stdout",
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

func exportSnapshot(output io.Writer, snapName string) *probe.Error {
	snapFile, perr := getSnapsFile(snapName)
	if perr != nil {
		return perr
	}
	f, err := os.Open(snapFile)
	if err != nil {
		return probe.NewError(err)
	}
	defer f.Close()
	_, err = io.Copy(output, f)
	return probe.NewError(err)
}

// main entry point for snapshot create.
func mainSnapExport(ctx *cli.Context) error {

	// Validate command-line args.
	snapName := parseSnapExportSyntax(ctx)

	// Create a snapshot.
	fatalIf(exportSnapshot(os.Stdout, snapName).Trace(), "Unable to export the specified snapshot")
	return nil
}
