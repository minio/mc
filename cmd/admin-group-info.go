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

var adminGroupInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display group info",
	Action:       mainAdminGroupInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET GROUPNAME

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get info on group 'allcents'.
     {{.Prompt}} {{.HelpName}} myminio allcents
`,
}

// checkAdminGroupInfoSyntax - validate all the passed arguments
func checkAdminGroupInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}
}

// mainAdminGroupInfo is the handle for "mc admin group info" command.
func mainAdminGroupInfo(ctx *cli.Context) error {
	checkAdminGroupInfoSyntax(ctx)

	console.SetColor("GroupMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	group := args.Get(1)
	gd, err1 := client.GetGroupDescription(globalContext, group)
	fatalIf(probe.NewError(err1).Trace(args...), "Could not get group info")

	printMsg(groupMessage{
		op:          "info",
		GroupName:   group,
		GroupStatus: gd.Status,
		GroupPolicy: gd.Policy,
		Members:     gd.Members,
	})

	return nil
}
