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
	"fmt"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var batchGenerateCmd = cli.Command{
	Name:         "generate",
	Usage:        "generate a new batch job definition",
	Action:       mainBatchGenerate,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET JOBTYPE

JOBTYPE:
` + supportedJobTypes() + `
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Generate a new batch 'replication' job definition:
     {{.Prompt}} {{.HelpName}} myminio replicate > replication.yaml
`,
}

func supportedJobTypes() string {
	var builder strings.Builder
	for _, jobType := range madmin.SupportedJobTypes {
		builder.WriteString("  - ")
		builder.WriteString(string(jobType))
		builder.WriteString("\n")
	}
	return builder.String()
}

// checkBatchGenerateSyntax - validate all the passed arguments
func checkBatchGenerateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainBatchGenerate is the handle for "mc batch generate" command.
func mainBatchGenerate(ctx *cli.Context) error {
	checkBatchGenerateSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	jobType := args.Get(1)

	// Start a new MinIO Admin Client
	adminClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var found bool
	for _, job := range madmin.SupportedJobTypes {
		if jobType == string(job) {
			found = true
			break
		}
	}
	if !found {
		fatalIf(errInvalidArgument().Trace(jobType), "Unable to generate a job template for the specified job type")
	}

	out, e := adminClient.GenerateBatchJob(globalContext, madmin.GenerateBatchJobOpts{
		Type: madmin.BatchJobType(jobType),
	})
	fatalIf(probe.NewError(e), "Unable to generate %s", args.Get(1))

	fmt.Println(string(out))
	return nil
}
