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

// Snapshot command
var snapCmd = cli.Command{
	Name:            "snap",
	Usage:           "generate snapshots for commands",
	Action:          mainSnap,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	Subcommands: []cli.Command{
		snapCreate,
		snapExport,
		snapImport,
		snapList,
		snapRemove,
	},
}

// snapshotPrefix is prepended to a snapshot so the client can identify it.
const snapshotPrefix = "snap://"

// isSnapsDirExists - verify if snaps directory exists.
func isSnapsDirExists() bool {
	SnapsDir, err := getSnapsDir()
	fatalIf(err.Trace(), "Unable to determine snaps folder.")
	if _, e := os.Stat(SnapsDir); e != nil {
		return false
	}
	return true
}

// getSnapsDir - return the full path of snaps dir
func getSnapsDir() (string, *probe.Error) {
	p, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}
	return filepath.Join(p, globalMCSnapsDir), nil
}

// createSnapsDir - create MinIO Client snaps folder
func createSnapsDir() *probe.Error {
	p, err := getSnapsDir()
	if err != nil {
		return err.Trace()
	}
	if e := os.MkdirAll(p, 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

func mainSnap(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
}
