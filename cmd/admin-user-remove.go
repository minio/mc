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

var adminUserRemoveCmd = cli.Command{
	Name:         "remove",
	Usage:        "remove user",
	Action:       mainAdminUserRemove,
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
  1. Remove a user 'foobar' on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio foobar
`,
}

// checkAdminUserRemoveSyntax - validate all the passed arguments
func checkAdminUserRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
}

// mainAdminUserRemove is the handle for "mc admin user remove" command.
func mainAdminUserRemove(ctx *cli.Context) error {
	checkAdminUserRemoveSyntax(ctx)

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.RemoveUser(globalContext, args.Get(1))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to remove %s", args.Get(1))

	printMsg(userMessage{
		op:        "remove",
		AccessKey: args.Get(1),
	})

	return nil
}
