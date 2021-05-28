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
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminGroupRemoveCmd = cli.Command{
	Name:         "remove",
	Usage:        "remove group or members from a group",
	Action:       mainAdminGroupRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET GROUPNAME [USERNAMES...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove members 'tencent' and 'fivecent' from group 'allcents'.
     {{.Prompt}} {{.HelpName}} myminio allcents tencent fivecent

  2. Remove group 'allcents'.
     {{.Prompt}} {{.HelpName}} myminio allcents
`,
}

// checkAdminGroupRemoveSyntax - validate all the passed arguments
func checkAdminGroupRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
}

// mainAdminGroupRemove is the handle for "mc admin group remove" command.
func mainAdminGroupRemove(ctx *cli.Context) error {
	checkAdminGroupRemoveSyntax(ctx)

	console.SetColor("GroupMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	members := []string{}
	for i := 2; i < ctx.NArg(); i++ {
		members = append(members, args.Get(i))
	}
	gAddRemove := madmin.GroupAddRemove{
		Group:    args.Get(1),
		Members:  members,
		IsRemove: true,
	}

	e := client.UpdateGroupMembers(globalContext, gAddRemove)
	fatalIf(probe.NewError(e).Trace(args...), "Could not perform remove operation")

	printMsg(groupMessage{
		op:        "remove",
		GroupName: args.Get(1),
		Members:   members,
	})

	return nil
}
