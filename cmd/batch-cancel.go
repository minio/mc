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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var batchCancelFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "id",
		Usage: "job id",
	},
}

var batchCancelCmd = cli.Command{
	Name:         "cancel",
	Usage:        "cancel ongoing batch job",
	Action:       mainBatchCancel,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(batchCancelFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET JOBFILE

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Cancel ongoing batch job:
     {{.Prompt}} {{.HelpName}} myminio <job-id>
`,
}

// batchCancelMessage container for file batchCancel messages
type batchCancelMessage struct {
	Status string `json:"status"`
	JobID  string `json:"job-id"`
}

// String colorized batchCancel message
func (c batchCancelMessage) String() string {
	return console.Colorize("batchCancel", fmt.Sprintf("Successfully canceled batch job `%s`", c.JobID))
}

// JSON jsonified batchCancel message
func (c batchCancelMessage) JSON() string {
	c.Status = "success"
	batchCancelMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(batchCancelMessageBytes)
}

// checkBatchCancelSyntax - validate all the passed arguments
func checkBatchCancelSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainBatchCancel is the handle for "mc batch cancel" command.
func mainBatchCancel(ctx *cli.Context) error {
	checkBatchCancelSyntax(ctx)

	console.SetColor("BatchCancel", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	jobID := args.Get(1)
	// Start a new MinIO Admin Client
	adminClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")
	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	e := adminClient.CancelBatchJob(ctxt, jobID)
	fatalIf(probe.NewError(e), "Unable to cancel job")

	printMsg(batchCancelMessage{
		Status: "Canceled",
		JobID:  jobID,
	})
	return nil
}
