/*
 * Minio Client (C) 2018 Minio, Inc.
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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var adminUsersPolicyCmd = cli.Command{
	Name:   "policy",
	Usage:  "Set policy for user",
	Action: mainAdminUsersPolicy,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET USERNAME POLICYNAME

POLICYNAME:
  Name of the canned policy created on Minio server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set a policy 'writeonly' to 'newuser' on Minio server.
     $ {{.HelpName}} myminio newuser writeonly
`,
}

// checkAdminUsersPolicySyntax - validate all the passed arguments
func checkAdminUsersPolicySyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code
	}
}

// mainAdminUsersPolicy is the handle for "mc admin users policy" command.
func mainAdminUsersPolicy(ctx *cli.Context) error {
	checkAdminUsersPolicySyntax(ctx)

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	fatalIf(probe.NewError(client.SetUserPolicy(args.Get(1), args.Get(2))).Trace(args...), "Cannot set user policy for user")

	printMsg(userMessage{
		op:         "policy",
		AccessKey:  args.Get(1),
		PolicyName: args.Get(2),
		UserStatus: "enabled",
	})

	return nil
}
