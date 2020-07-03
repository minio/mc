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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	snapRemoveFlags = []cli.Flag{}
)

var snapRemove = cli.Command{
	Name:   "remove",
	Usage:  "Remove a specific snapshot",
	Action: mainSnapRemove,
	Before: setGlobalsFromContext,
	Flags:  append(snapRemoveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} COMMAND - {{.Usage}}

USAGE:
  {{.HelpName}} SNAPSHOT-NAME

EXAMPLES:
  1. Remove a snapshot from the local machine
      {{.Prompt}} {{.HelpName}} my-snapshot-name
`,
}

// Validate command-line args.
func parseSnapRemoveSyntax(ctx *cli.Context) string {
	return ctx.Args().Get(0)
}

func removeSnapshot(snapName string) *probe.Error {
	snapFile, err := getSnapsFile(snapName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(snapFile); err != nil {
		return probe.NewError(err)
	}

	e := os.Remove(snapFile)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Main entry point for snapshot list
func mainSnapRemove(ctx *cli.Context) error {
	// Validate command-line args.
	snapshotName := parseSnapRemoveSyntax(ctx)

	fatalIf(removeSnapshot(snapshotName).Trace(), "Unable to remove the specified snapshot")
	return nil
}
