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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var batchDescribeCmd = cli.Command{
	Name:         "describe",
	Usage:        "describe job definition for a job",
	Action:       mainBatchDescribe,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET JOBID

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Describe current batch job definition:
     {{.Prompt}} {{.HelpName}} myminio KwSysDpxcBU9FNhGkn2dCf
`,
}

// checkBatchDescribeSyntax - validate all the passed arguments
func checkBatchDescribeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainBatchDescribe is the handle for "mc batch create" command.
func mainBatchDescribe(ctx *cli.Context) error {
	checkBatchDescribeSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	jobID := args.Get(1)

	// Start a new MinIO Admin Client
	adminClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	job, e := adminClient.DescribeBatchJob(ctxt, jobID)
	fatalIf(probe.NewError(e), "Unable to fetch the job definition")

	fmt.Println(job)
	return nil
}
