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

var adminUserListCmd = cli.Command{
	Name:         "list",
	Usage:        "list all users",
	Action:       mainAdminUserList,
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
  1. List all users on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminUserListSyntax - validate all the passed arguments
func checkAdminUserListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// mainAdminUserList is the handle for "mc admin user list" command.
func mainAdminUserList(ctx *cli.Context) error {
	checkAdminUserListSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("UserMessage", color.New(color.FgGreen))
	console.SetColor("AccessKey", color.New(color.FgBlue))
	console.SetColor("PolicyName", color.New(color.FgYellow))
	console.SetColor("UserStatus", color.New(color.FgCyan))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	users, e := client.ListUsers(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to list user")

	for k, v := range users {
		printMsg(userMessage{
			op:         "list",
			AccessKey:  k,
			PolicyName: v.PolicyName,
			UserStatus: string(v.Status),
		})
	}
	return nil
}
