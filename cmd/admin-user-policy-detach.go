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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminUserPolicyDetachCmd = cli.Command{
	Name:         "detach",
	Usage:        "detach an IAM policy from a user",
	Action:       mainAdminUserPolicyDetach,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME USERNAME

POLICYNAME:
  Name of the policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Detach the "diagnostics" policy from user "james".
     {{.Prompt}} {{.HelpName}} myminio diagnostics james
`,
}

func checkAdminUserPolicyDetachSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		showCommandHelpAndExit(ctx, "detach", 1) // last argument is exit code
	}
}

// mainAdminUserPolicyDetach is the handler for "mc admin group policy detach" command.
func mainAdminUserPolicyDetach(ctx *cli.Context) error {
	checkAdminUserPolicyDetachSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policiesToDetach := args.Get(1)
	user := args.Get(2)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.DetachPoliciesFromUser(globalContext, policiesToDetach, user)
	if e == nil {
		printMsg(userPolicyMessage{
			op:          "detach",
			Policy:      policiesToDetach,
			UserOrGroup: user,
			IsGroup:     false,
		})
	} else {
		fatalIf(probe.NewError(e), "Unable to detach the policy")
	}
	return nil
}
