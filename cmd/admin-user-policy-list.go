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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminUserPolicyListCmd = cli.Command{
	Name:         "list",
	Usage:        "list all IAM policies attached to a user",
	Action:       mainAdminUserPolicyList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET USERNAME

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all policies attached to user 'james'.
     {{.Prompt}} {{.HelpName}} myminio james
`,
}

// checkAdminPolicyListSyntax - validate all the passed arguments
func checkAdminUserPolicyListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// mainAdminPolicyList is the handle for "mc admin policy add" command.
func mainAdminUserPolicyList(ctx *cli.Context) error {
	checkAdminUserPolicyListSyntax(ctx)

	console.SetColor("PolicyName", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	policies, e := client.GetUserPolicies(globalContext, user)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get user policy info")

	for _, k := range policies {
		printMsg(userPolicyMessage{
			op:     "list",
			Policy: k,
		})
	}
	return nil
}
