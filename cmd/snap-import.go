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
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	snapImportFlags = []cli.Flag{}
)

// FIXME:
var snapImport = cli.Command{
	Name:   "import",
	Usage:  "Import a snapshot from JSON snapshot archive",
	Action: mainSnapImport,
	Before: setGlobalsFromContext,
	Flags:  append(snapImportFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} COMMAND - {{.Usage}}

USAGE:
  {{.HelpName}} COMMAND

COMMAND:

EXAMPLES:
`,
}

// validate command-line args.
func parseSnapImportSyntax(ctx *cli.Context) (snapName string) {
	args := ctx.Args()
	if len(args) != 1 {
		// fatalIf(errors.New("wrong arguments"), "")
	}

	return args.Get(0)
}

func importSnapshot(input io.Reader, snapName string) *probe.Error {
	snapsDir, err := getSnapsDir()
	if err != nil {
		return err
	}

	snapDir := filepath.Join(snapsDir, snapName)
	if _, e := os.Stat(snapDir); e == nil {
		return probe.NewError(errors.New("snapshot already exist"))
	} else {
		if !os.IsNotExist(e) {
			return probe.NewError(e)
		}
	}

	e := os.Mkdir(snapDir, 0700)
	if e != nil {
		return probe.NewError(e)
	}

	e = decompress(os.Stdin, snapDir)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// main entry point for snapshot import
func mainSnapImport(ctx *cli.Context) error {
	// Validate command-line args.
	snapName := parseSnapExportSyntax(ctx)

	// Create a snapshot.
	err := importSnapshot(os.Stdin, snapName)
	fatalIf(err.Trace(), "Unable to import the specified snapshot")
	return nil
}
