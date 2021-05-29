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

var adminPolicyInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "show info on a policy",
	Action:       mainAdminPolicyInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME

POLICYNAME:
  Name of the policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show information on a given policy.
     {{.Prompt}} {{.HelpName}} myminio writeonly
`,
}

// checkAdminPolicyInfoSyntax - validate all the passed arguments
func checkAdminPolicyInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}
}

// mainAdminPolicyInfo is the handler for "mc admin policy info" command.
func mainAdminPolicyInfo(ctx *cli.Context) error {
	checkAdminPolicyInfoSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policyName := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection")

	buf, e := client.InfoCannedPolicy(globalContext, policyName)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch policy")

	printMsg(userPolicyMessage{
		op:         "info",
		Policy:     policyName,
		PolicyJSON: buf,
	})

	return nil
}
