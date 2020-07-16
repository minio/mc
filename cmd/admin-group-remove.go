/*
 * MinIO Client (C) 2019 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminGroupRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove group or members from a group",
	Action: mainAdminGroupRemove,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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
