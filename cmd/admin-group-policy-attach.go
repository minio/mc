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

var adminGroupPolicyAttachCmd = cli.Command{
	Name:         "attach",
	Usage:        "attach an IAM policy to a group",
	Action:       mainAdminGroupPolicyAttach,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME GROUPNAME

POLICYNAME:
  Name of the policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Attach the "diagnostics" policy to the group "auditors".
     {{.Prompt}} {{.HelpName}} myminio diagnostics auditors

  2. Add user "james" to group "staff", then add the "readwrite" policy to the group "staff".
     {{.Prompt}} mc admin group add myminio staff james
     {{.Prompt}} {{.HelpName}} myminio readwrite staff
`,
}

func checkAdminGroupPolicyAttachSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		showCommandHelpAndExit(ctx, "attach", 1) // last argument is exit code
	}
}

// mainAdminGroupPolicyAttach is the handler for "mc admin group policy attach" command.
func mainAdminGroupPolicyAttach(ctx *cli.Context) error {
	checkAdminGroupPolicyAttachSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policiesToAttach := args.Get(1)
	group := args.Get(2)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.AttachPoliciesToGroup(globalContext, policiesToAttach, group)
	if e == nil {
		printMsg(userPolicyMessage{
			op:          "attach",
			Policy:      policiesToAttach,
			UserOrGroup: group,
			IsGroup:     true,
		})
	} else {
		fatalIf(probe.NewError(e), "Unable to attach the policy")
	}
	return nil
}
