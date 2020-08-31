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

var adminPolicyRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove policy",
	Action: mainAdminPolicyRemove,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME

POLICYNAME:
  Name of the canned policy on MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove 'writeonly' policy on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio writeonly
`,
}

// checkAdminPolicyRemoveSyntax - validate all the passed arguments
func checkAdminPolicyRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
}

// mainAdminPolicyRemove is the handle for "mc admin policy remove" command.
func mainAdminPolicyRemove(ctx *cli.Context) error {
	checkAdminPolicyRemoveSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	fatalIf(probe.NewError(client.RemoveCannedPolicy(globalContext, args.Get(1))).Trace(args...), "Unable to remove policy")

	printMsg(userPolicyMessage{
		op:     "remove",
		Policy: args.Get(1),
	})

	return nil
}
