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

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/console"
)

var replicateStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "show server side replication status",
	Action:       mainReplicateStatus,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} TARGET

FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
EXAMPLES:
  1. Get server side replication metrics for bucket "mybucket" for alias "myminio".
	   {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkReplicateStatusSyntax - validate all the passed arguments
func checkReplicateStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "status", 1) // last argument is exit code
	}
}

type replicateStatusMessage struct {
	Op                string              `json:"op"`
	URL               string              `json:"url"`
	Status            string              `json:"status"`
	ReplicationStatus replication.Metrics `json:"replicationStatus"`
}

func (s replicateStatusMessage) JSON() string {
	s.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func printReplicateStatusHeader() {
	if globalJSON {
		return
	}
	maxLen := 15
	console.Println(console.Colorize("Headers", newPrettyTable(" | ",
		Field{"Status", 20},
		Field{"Size", maxLen},
		Field{"Count", maxLen},
	).buildRow("Replication Status", "Size (Bytes)", "Count")))
}

func (s replicateStatusMessage) String() string {
	maxLen := 15
	var contents = [][]string{
		{"Pending", humanize.IBytes(s.ReplicationStatus.PendingSize), humanize.Comma(int64(s.ReplicationStatus.PendingCount))},
		{"Failed", humanize.IBytes(s.ReplicationStatus.FailedSize), humanize.Comma(int64(s.ReplicationStatus.FailedCount))},
		{"Replicated", humanize.IBytes(s.ReplicationStatus.ReplicatedSize), ""},
		{"Replica", humanize.IBytes(s.ReplicationStatus.ReplicaSize), ""},
	}
	var rows string
	var theme = []string{"Pending", "Failed", "Replica", "Replica"}
	for i, row := range contents {
		th := theme[i]
		if row[1] == "0 B" && i == 1 {
			th = theme[0]
		}
		r := console.Colorize(th, newPrettyTable(" | ",
			Field{"Status", 20},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow(row[0], row[1], row[2])+"\n")
		rows += r
	}
	return console.Colorize("replicateStatusMessage", rows)
}

func mainReplicateStatus(cliCtx *cli.Context) error {
	ctx, cancelReplicateStatus := context.WithCancel(globalContext)
	defer cancelReplicateStatus()

	console.SetColor("Headers", color.New(color.FgGreen))
	console.SetColor("Replica", color.New(color.Bold, color.FgCyan))
	console.SetColor("Failed", color.New(color.Bold, color.FgRed))
	console.SetColor("Pending", color.New(color.Bold, color.FgWhite))

	checkReplicateStatusSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	replicateStatus, err := client.GetReplicationMetrics(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication status")

	printReplicateStatusHeader()
	printMsg(replicateStatusMessage{
		Op:                "status",
		URL:               aliasedURL,
		ReplicationStatus: replicateStatus,
	})

	return nil
}
