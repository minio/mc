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
	"context"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	snapListFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "file, f",
			Usage: "Use the snapshot file",
		},
	}
)

const snapshotSuffix = ".snap"

var snapList = cli.Command{
	Name:   "list",
	Usage:  "List all snapshots descriptions stored locally",
	Action: mainSnapList,
	Before: setGlobalsFromContext,
	Flags:  append(snapListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all created snapshots
     {{.Prompt}} {{.HelpName}}

  2. List the contents of a snapshot
     {{.Prompt}} {{.HelpName}} my-snapshot-name

  3. List the contents of a snapshot file stored in the local machine
     {{.Prompt}} {{.HelpName}} -f /path/to/my-snapshot.snap

`,
}

// listSnapMsg container for snap creation message structure
type listSnapMsg struct {
	Status       string    `json:"success"`
	SnapshotName string    `json:"snapshot"`
	ModTime      time.Time `json:"modTime"`
}

func (r listSnapMsg) String() string {
	return console.Colorize("Time", "["+r.ModTime.Round(time.Second).String()+"]") + " " + r.SnapshotName
}

func (r listSnapMsg) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// Validate command-line args.
func parseSnapListSyntax(ctx *cli.Context) string {
	return cleanSnapName(ctx.Args().First())
}

func listSnapshots() ([]os.FileInfo, *probe.Error) {
	snapsDir, err := getSnapsDir()
	if err != nil {
		return nil, err
	}

	entries, e := ioutil.ReadDir(snapsDir)
	if e != nil {
		return nil, probe.NewError(e)
	}

	var snapshots []os.FileInfo
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), snapshotSuffix) {
			continue
		}
		snapshots = append(snapshots, entry)
	}

	return snapshots, nil
}

// Main entry point for snapshot list
func mainSnapList(cmdCtx *cli.Context) error {
	// Validate command-line args.
	snapshot := parseSnapListSyntax(cmdCtx)

	// Additional command specific theme customization.
	console.SetColor("File", color.New(color.Bold))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("Time", color.New(color.FgGreen))

	if snapshot == "" {
		snapshots, err := listSnapshots()
		fatalIf(err.Trace(), "Unable to list snapshots")
		for _, s := range snapshots {
			name := strings.TrimSuffix(s.Name(), snapshotSuffix)
			printMsg(listSnapMsg{SnapshotName: name, ModTime: s.ModTime()})
		}
		return nil
	}

	var (
		clnt Client
		err  *probe.Error
	)

	if cmdCtx.Bool("file") {
		// We are going to list a snapshot file in the local machine
		f, e := os.Open(snapshot)
		if e != nil {
			err = probe.NewError(e)
		} else {
			clnt, err = newSnapClientReader("dummy-alias", "dummy-alias/", f)
		}
	} else {
		clnt, err = newClient(snapshotPrefix + snapshot)
	}

	fatalIf(err.Trace(), "Unable to list snapshot")

	ctx, cancelList := context.WithCancel(globalContext)
	defer cancelList()
	return doList(ctx, clnt, true, false, time.Time{}, true)
}
