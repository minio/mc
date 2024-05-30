// Copyright (c) 2015-2022 MinIO, Inc.
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
	"github.com/minio/pkg/v3/console"
)

var replicateResyncStatusFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "remote-bucket",
		Usage: "remote bucket ARN",
	},
}

var replicateResyncStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "status of replication recovery",
	Action:       mainreplicateResyncStatus,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateResyncStatusFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} TARGET

FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
EXAMPLES:
  1. Status of replication resync in bucket "mybucket" under alias "myminio" for all targets.
   {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. Status of replication resync in bucket "mybucket" under specific remote bucket target.
   {{.Prompt}} {{.HelpName}} myminio/mybucket --remote-bucket "arn:minio:replication::xxx:mybucket"
`,
}

// checkreplicateResyncStatusSyntax - validate all the passed arguments
func checkreplicateResyncStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type replicateResyncStatusMessage struct {
	Op                string                        `json:"op"`
	URL               string                        `json:"url"`
	ResyncTargetsInfo replication.ResyncTargetsInfo `json:"resyncInfo"`
	Status            string                        `json:"status"`
	TargetArn         string                        `json:"targetArn"`
}

func (r replicateResyncStatusMessage) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r replicateResyncStatusMessage) String() string {
	if len(r.ResyncTargetsInfo.Targets) == 0 {
		return console.Colorize("replicateResyncStatusWarn", "No replication resync status available.")
	}
	coloredDot := console.Colorize("Headers", dot)
	var rows string
	rows += console.Colorize("TDetail", "Resync status summary:")

	for _, st := range r.ResyncTargetsInfo.Targets {
		rows += "\n"
		rows += console.Colorize("replicateResyncStatusMsg", newPrettyTable(" | ",
			Field{"ARN", 120},
		).buildRow(fmt.Sprintf("%s %s", coloredDot, st.Arn)))
		rows += "\n"
		rows += console.Colorize("TDetail", "   Status: ")
		rows += console.Colorize(st.ResyncStatus, st.ResyncStatus)
		rows += "\n"

		maxLen := 15
		theme := []string{"Replicated", "Failed"}
		rows += console.Colorize("THeaders", newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("   Replication Status", "Size (Bytes)", "Count"))
		rows += "\n"
		rows += console.Colorize(theme[0], newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("   Replicated", humanize.IBytes(uint64(st.ReplicatedSize)), humanize.Comma(int64(st.ReplicatedCount))))
		rows += "\n"
		rows += console.Colorize(theme[0], newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("   Failed", humanize.IBytes(uint64(st.FailedSize)), humanize.Comma(int64(st.FailedCount))))
		rows += "\n"
	}
	return rows
}

func mainreplicateResyncStatus(cliCtx *cli.Context) error {
	ctx, cancelreplicateResyncStatus := context.WithCancel(globalContext)
	defer cancelreplicateResyncStatus()

	console.SetColor("replicateResyncStatusWarn", color.New(color.FgHiYellow))
	console.SetColor("replicateResyncStatusMsg", color.New(color.FgGreen))
	console.SetColor("Headers", color.New(color.FgGreen, color.Bold))
	console.SetColor("THeaders", color.New(color.Bold, color.FgCyan))

	console.SetColor("TDetail", color.New(color.FgWhite, color.Bold))
	console.SetColor("Ongoing", color.New(color.Bold, color.FgYellow))
	console.SetColor("Failed", color.New(color.Bold, color.FgRed))
	console.SetColor("Completed", color.New(color.Bold, color.FgGreen))

	checkreplicateResyncStatusSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")

	rinfo, err := client.ReplicationResyncStatus(ctx, cliCtx.String("remote-bucket"))
	fatalIf(err.Trace(args...), "Unable to get replication resync status")
	printMsg(replicateResyncStatusMessage{
		Op:                cliCtx.Command.Name,
		URL:               aliasedURL,
		ResyncTargetsInfo: rinfo,
		TargetArn:         cliCtx.String("remote-bucket"),
	})
	return nil
}
