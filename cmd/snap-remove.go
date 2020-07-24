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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	snapRemoveFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "Force removing the snapshot without a prompt",
		},
	}
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

// removeSnapMsg container for snap creation message structure
type removeSnapMsg struct {
	Status       string `json:"success"`
	SnapshotName string `json:"snapshot"`
}

func (r removeSnapMsg) String() string {
	return console.Colorize("SnapDeletion", "The snapshot `"+r.SnapshotName+"` is removed.")
}

func (r removeSnapMsg) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// Validate command-line args.
func parseSnapRemoveSyntax(ctx *cli.Context) (string, bool) {
	args := ctx.Args()
	if len(args) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "remove", globalErrorExitStatus)
	}

	snapshotName := ctx.Args().Get(0)
	snapshotName = filepath.ToSlash(snapshotName)
	snapshotName = strings.TrimRight(snapshotName, "/")
	return snapshotName, ctx.Bool("force")
}

func removeSnapshot(snapName string, force bool) *probe.Error {
	approved, answered := false, false
	isTerminal := terminal.IsTerminal(int(os.Stdin.Fd()))
	if !force && !globalJSON && !globalQuiet && isTerminal {
		for !answered {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Are you sure you want to remove this snapshot ? [y/n] ")
			value, _, _ := reader.ReadLine()
			switch strings.ToLower(string(value)) {
			case "y", "yes":
				approved = true
				answered = true
			case "n", "no":
				answered = true
			}
		}
	} else {
		approved = true
	}

	if !approved {
		return probe.NewError(errors.New("user declined snapshot deletion"))
	}

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

	console.SetColor("SnapDeletion", color.New(color.FgGreen))

	// Validate command-line args.
	snapshotName, force := parseSnapRemoveSyntax(ctx)
	fatalIf(removeSnapshot(snapshotName, force).Trace(), "Unable to remove the specified snapshot")

	printMsg(removeSnapMsg{Status: "success", SnapshotName: snapshotName})
	return nil
}
