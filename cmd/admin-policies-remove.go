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

var adminPoliciesRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "Remove policies",
	Action: mainAdminPoliciesRemove,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME

POLICYNAME:
  Name of the canned policy on Minio server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove a canned policy 'writeonly'.
     $ {{.HelpName}} myminio writeonly
`,
}

// checkAdminPoliciesRemoveSyntax - validate all the passed arguments
func checkAdminPoliciesRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
}

// mainAdminPoliciesRemove is the handle for "mc admin users add" command.
func mainAdminPoliciesRemove(ctx *cli.Context) error {
	checkAdminPoliciesRemoveSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	fatalIf(probe.NewError(client.RemoveCannedPolicy(args.Get(1))).Trace(args...), "Cannot remove policy")

	printMsg(userPolicyMessage{
		op:     "remove",
		Policy: args.Get(1),
	})

	return nil
}
