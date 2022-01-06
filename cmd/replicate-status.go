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
	"fmt"

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

func (s replicateStatusMessage) String() string {
	coloredDot := console.Colorize("Headers", dot)
	maxLen := 15
	var contents [][]string

	var rows string
	arntheme := []string{"Headers"}
	theme := []string{"Pending", "Failed", "Replicated", "Replica"}
	contents = append(contents, []string{"Pending", humanize.IBytes(s.ReplicationStatus.PendingSize), humanize.Comma(int64(s.ReplicationStatus.PendingCount))})
	contents = append(contents, []string{"Failed", humanize.IBytes(s.ReplicationStatus.FailedSize), humanize.Comma(int64(s.ReplicationStatus.FailedCount))})
	contents = append(contents, []string{"Replicated", humanize.IBytes(s.ReplicationStatus.ReplicatedSize), ""})
	contents = append(contents, []string{"Replica", humanize.IBytes(s.ReplicationStatus.ReplicaSize), ""})
	var th string

	if s.ReplicationStatus.PendingSize == 0 &&
		s.ReplicationStatus.FailedSize == 0 &&
		s.ReplicationStatus.ReplicaSize == 0 &&
		s.ReplicationStatus.ReplicatedSize == 0 {
		return "Replication status not available."
	}
	r := console.Colorize("THeaders", newPrettyTable(" | ",
		Field{"Summary", 95},
	).buildRow("Summary: "))
	rows += r
	rows += "\n"

	hIdx := 0
	for i, row := range contents {
		if i%4 == 0 {
			if hIdx > 0 {
				rows += "\n"
			}
			hIdx++
			rows += console.Colorize("TgtHeaders", newPrettyTable(" | ",
				Field{"Status", 21},
				Field{"Size", maxLen},
				Field{"Count", maxLen},
			).buildRow("Replication Status   ", "Size (Bytes)", "Count"))
			rows += "\n"
		}

		idx := i % 4
		th = theme[idx]
		r := console.Colorize(th, newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("   "+row[0], row[1], row[2])+"\n")
		rows += r
	}

	contents = nil
	var arns []string
	for arn := range s.ReplicationStatus.Stats {
		arns = append(arns, arn)
	}
	for _, st := range s.ReplicationStatus.Stats {
		contents = append(contents, []string{"Pending", humanize.IBytes(st.PendingSize), humanize.Comma(int64(st.PendingCount))})
		contents = append(contents, []string{"Failed", humanize.IBytes(st.FailedSize), humanize.Comma(int64(st.FailedCount))})
		contents = append(contents, []string{"Replicated", humanize.IBytes(st.ReplicatedSize), ""})
	}
	if len(contents) > 0 {
		rows += "\n"
		r := console.Colorize("THeaders", newPrettyTable(" | ",
			Field{"Target statuses", 95},
		).buildRow("Remote Target Statuses: "))
		rows += r
		rows += "\n"
	}
	hIdx = 0
	for i, row := range contents {
		if i%3 == 0 {
			if hIdx > 0 {
				rows += "\n"
			}
			th = arntheme[0]
			r := console.Colorize(th, newPrettyTable(" | ",
				Field{"ARN", 120},
			).buildRow(fmt.Sprintf("%s %s", coloredDot, arns[hIdx])))
			rows += r
			rows += "\n"
			hIdx++
			rows += console.Colorize("TgtHeaders", newPrettyTable(" | ",
				Field{"Status", 21},
				Field{"Size", maxLen},
				Field{"Count", maxLen},
			).buildRow("Replication Status   ", "Size (Bytes)", "Count"))
			rows += "\n"
		}

		idx := i % 3
		th = theme[idx]
		r := console.Colorize(th, newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("   "+row[0], row[1], row[2])+"\n")
		rows += r
	}
	return console.Colorize("replicateStatusMessage", rows)
}

func mainReplicateStatus(cliCtx *cli.Context) error {
	ctx, cancelReplicateStatus := context.WithCancel(globalContext)
	defer cancelReplicateStatus()

	console.SetColor("THeaders", color.New(color.Bold, color.FgHiWhite))
	console.SetColor("Headers", color.New(color.Bold, color.FgGreen))
	console.SetColor("TgtHeaders", color.New(color.Bold, color.FgCyan))

	console.SetColor("Replica", color.New(color.FgCyan))
	console.SetColor("Failed", color.New(color.Bold, color.FgRed))
	console.SetColor("Pending", color.New(color.FgWhite))

	checkReplicateStatusSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	replicateStatus, err := client.GetReplicationMetrics(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication status")

	printMsg(replicateStatusMessage{
		Op:                "status",
		URL:               aliasedURL,
		ReplicationStatus: replicateStatus,
	})

	return nil
}
