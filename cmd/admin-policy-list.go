/*
 * MinIO Client (C) 2018-2019 MinIO, Inc.
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

var adminPolicyListCmd = cli.Command{
	Name:   "list",
	Usage:  "list all policies",
	Action: mainAdminPolicyList,
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
  1. List all policies on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminPolicyListSyntax - validate all the passed arguments
func checkAdminPolicyListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// mainAdminPolicyList is the handle for "mc admin policy add" command.
func mainAdminPolicyList(ctx *cli.Context) error {
	checkAdminPolicyListSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	policies, e := client.ListCannedPolicies(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to list policy")

	for k := range policies {
		printMsg(userPolicyMessage{
			op:     "list",
			Policy: k,
		})
	}
	return nil
}
