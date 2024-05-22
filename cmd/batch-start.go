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
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var batchStartCmd = cli.Command{
	Name:         "start",
	Usage:        "start a new batch job",
	Action:       mainBatchStart,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET JOBFILE

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Start a new batch 'replication' job:
     {{.Prompt}} {{.HelpName}} myminio ./replication.yaml
`,
}

// batchStartMessage container for file batchStart messages
type batchStartMessage struct {
	Status string                `json:"status"`
	Result madmin.BatchJobResult `json:"result"`
}

// String colorized batchStart message
func (c batchStartMessage) String() string {
	return console.Colorize("BatchStart", fmt.Sprintf("Successfully started '%s' job `%s` on '%s'", c.Result.Type, c.Result.ID, c.Result.Started))
}

// JSON jsonified batchStart message
func (c batchStartMessage) JSON() string {
	c.Status = "success"
	batchStartMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(batchStartMessageBytes)
}

// checkBatchStartSyntax - validate all the passed arguments
func checkBatchStartSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainBatchStart is the handle for "mc batch create" command.
func mainBatchStart(ctx *cli.Context) error {
	checkBatchStartSyntax(ctx)

	console.SetColor("BatchStart", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Start a new MinIO Admin Client
	adminClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	buf, e := os.ReadFile(args.Get(1))
	fatalIf(probe.NewError(e), "Unable to read %s", args.Get(1))

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	res, e := adminClient.StartBatchJob(ctxt, string(buf))
	fatalIf(probe.NewError(e), "Unable to start job")

	printMsg(batchStartMessage{
		Status: "success",
		Result: res,
	})
	return nil
}
