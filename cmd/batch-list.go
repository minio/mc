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
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/olekukonko/tablewriter"
)

var batchListFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "type",
		Usage: "list all current batch jobs via job type",
	},
}

var batchListCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "list all current batch jobs",
	Action:       mainBatchList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(batchListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all current batch jobs:
     {{.Prompt}} {{.HelpName}} myminio

  2. List all current batch jobs of type 'replicate':
     {{.Prompt}} {{.HelpName}} myminio/ --type "replicate"
`,
}

// batchListMessage container for file batchList messages
type batchListMessage struct {
	Status string                  `json:"status"`
	Jobs   []madmin.BatchJobResult `json:"jobs"`
}

// String colorized batchList message
func (c batchListMessage) String() string {
	if len(c.Jobs) == 0 {
		return "currently no jobs are running"
	}

	var s strings.Builder

	// Set table header
	table := tablewriter.NewWriter(&s)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	table.SetHeader([]string{"ID", "TYPE", "USER", "STARTED"})
	data := make([][]string, 0, 4)

	for _, job := range c.Jobs {
		data = append(data, []string{
			job.ID,
			string(job.Type),
			job.User,
			humanize.Time(job.Started),
		})
	}

	table.AppendBulk(data)
	table.Render()

	return s.String()
}

// JSON jsonified batchList message
func (c batchListMessage) JSON() string {
	c.Status = "success"
	batchListMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(batchListMessageBytes)
}

// checkBatchListSyntax - validate all the passed arguments
func checkBatchListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainBatchList is the handle for "mc batch create" command.
func mainBatchList(ctx *cli.Context) error {
	checkBatchListSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Start a new MinIO Admin Client
	adminClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	res, e := adminClient.ListBatchJobs(ctxt, &madmin.ListBatchJobsFilter{
		ByJobType: ctx.String("type"),
	})
	fatalIf(probe.NewError(e), "Unable to list jobs")

	printMsg(batchListMessage{
		Status: "success",
		Jobs:   res.Jobs,
	})
	return nil
}
