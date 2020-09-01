/*
 * MinIO Client (C) 2018 MinIO, Inc.
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
)

var adminUserListCmd = cli.Command{
	Name:   "list",
	Usage:  "list all users",
	Action: mainAdminUserList,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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
