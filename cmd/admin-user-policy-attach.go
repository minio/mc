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

var adminUserPolicyAttachCmd = cli.Command{
	Name:         "attach",
	Usage:        "attach an IAM policy to a user",
	Action:       mainAdminUserPolicyAttach,
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
  1. Attach the "readonly" policy to user "james".
     {{.Prompt}} {{.HelpName}} myminio readonly james

  2. Create a new user "jerry", then attach the "readwrite" policy to that user.
     {{.Prompt}} mc admin user add myminio jerry
     {{.Prompt}} {{.HelpName}} myminio readwrite jerry
`,
}

func checkAdminUserPolicyAttachSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		showCommandHelpAndExit(ctx, "attach", 1) // last argument is exit code
	}
}

// mainAdminUserPolicyAttach is the handler for "mc admin user policy attach" command.
func mainAdminUserPolicyAttach(ctx *cli.Context) error {
	checkAdminUserPolicyAttachSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policiesToAttach := args.Get(1)
	user := args.Get(2)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	userInfo, e := client.GetUserInfo(globalContext, user)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get user policy info")
	existingPolicies := userInfo.PolicyName

	updatedPolicies, e := attachCannedPolicies(existingPolicies, policiesToAttach)
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to attach the policy")
	}

	e = client.SetPolicy(globalContext, updatedPolicies, user, false)
	if e == nil {
		printMsg(userPolicyMessage{
			op:          "attach",
			Policy:      policiesToAttach,
			UserOrGroup: user,
			IsGroup:     false,
		})
	} else {
		fatalIf(probe.NewError(e), "Unable to attach the policy")
	}
	return nil
}
