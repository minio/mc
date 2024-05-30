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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var quotaClearCmd = cli.Command{
	Name:         "clear",
	Usage:        "clear bucket quota",
	Action:       mainQuotaClear,
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
  1. Clear bucket quota configured for bucket "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkQuotaClearSyntax - validate all the passed arguments
func checkQuotaClearSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainQuotaClear is the handler for "mc quota clear" command.
func mainQuotaClear(ctx *cli.Context) error {
	checkQuotaClearSyntax(ctx)

	console.SetColor("QuotaMessage", color.New(color.FgGreen))
	console.SetColor("QuotaInfo", color.New(color.FgCyan))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	_, targetURL := url2Alias(args[0])
	if e := client.SetBucketQuota(globalContext, targetURL, &madmin.BucketQuota{}); e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to clear bucket quota config")
	}
	printMsg(quotaMessage{
		op:     ctx.Command.Name,
		Bucket: targetURL,
		Status: "success",
	})

	return nil
}
