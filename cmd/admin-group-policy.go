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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminGroupPolicySubcommands = []cli.Command{
	adminGroupPolicyAttachCmd,
	adminGroupPolicyDetachCmd,
}

var adminGroupPolicyCmd = cli.Command{
	Name:            "policy",
	Usage:           "manage policies relating to groups",
	Action:          mainAdminGroupPolicy,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     adminGroupPolicySubcommands,
	HideHelpCommand: true,
}

// mainAdminGroupPolicy is the handle for "mc admin group policy" command.
func mainAdminGroupPolicy(ctx *cli.Context) error {
	commandNotFound(ctx, adminPolicySubcommands)
	return nil
	// Sub-commands like "attach", "list" have their own main.
}

func checkPolicyAttachDetachSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 3 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func groupAttachOrDetachPolicy(ctx *cli.Context, attach bool) error {
	checkPolicyAttachDetachSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	group := args.Get(1)

	var policyList []string
	for i := 2; i < len(args); i++ {
		policyList = append(policyList, args.Get(i))
	}

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var e error
	if attach {
		e = client.AttachPolicyGroup(globalContext, group, policyList)
	} else {
		e = client.DetachPolicyGroup(globalContext, group, policyList)
	}

	if e == nil {
		printMsg(userPolicyMessage{
			op:          ctx.Command.Name,
			Policy:      strings.Join(policyList, ", "),
			UserOrGroup: group,
		})
	} else {
		if attach {
			fatalIf(probe.NewError(e).Trace(args...), "Unable to attach policy to group")
		} else {
			fatalIf(probe.NewError(e).Trace(args...), "Unable to detach policy from group")
		}
	}
	return nil
}
