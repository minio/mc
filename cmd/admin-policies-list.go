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

var adminPoliciesListCmd = cli.Command{
	Name:   "list",
	Usage:  "List policies",
	Action: mainAdminPoliciesList,
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
  1. List all policies.
     $ {{.HelpName}} myminio

  2. List only one policy.
     $ {{.HelpName}} myminio writeonly
`,
}

// checkAdminPoliciesListSyntax - validate all the passed arguments
func checkAdminPoliciesListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 1 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// mainAdminPoliciesList is the handle for "mc admin users add" command.
func mainAdminPoliciesList(ctx *cli.Context) error {
	checkAdminPoliciesListSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	policies, e := client.ListCannedPolicies()
	fatalIf(probe.NewError(e).Trace(args...), "Cannot list policies")

	if policyName := args.Get(1); policyName != "" {
		printMsg(userPolicyMessage{
			op:         "list",
			Policy:     policyName,
			PolicyJSON: policies[policyName],
		})
	} else {
		for k := range policies {
			printMsg(userPolicyMessage{
				op:     "list",
				Policy: k,
			})
		}
	}
	return nil
}
