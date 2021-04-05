/*
 * MinIO Client (C) 2021 MinIO, Inc.
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

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
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
