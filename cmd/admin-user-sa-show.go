/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var adminUserSAShowCmd = cli.Command{
	Name:   "show",
	Usage:  "show the credentials of the specified service account",
	Action: mainAdminUserSAShow,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET SERVICE-ACCOUNT-ACCESS-KEY

SERVICE-ACCOUNT-ACCESS-KEY:
  The access key of the service account.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show the credentials of the service account 'SKA762Z7UPIFS5OL1CO4'.
     {{.Prompt}} {{.HelpName}} myminio/ SKA762Z7UPIFS5OL1CO4
`,
}

func checkAdminUserSAShowSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "show", 1) // last argument is exit code
	}
}

func mainAdminUserSAShow(ctx *cli.Context) error {
	setSACommandColors()
	checkAdminUserSAShowSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	serviceAccountKey := args.Get(1)

	creds, e := client.GetServiceAccount(globalContext, serviceAccountKey)
	fatalIf(probe.NewError(e).Trace(args...), "Cannot show the credentials of the specified service account")

	printMsg(saMessage{
		AccessKey:    creds.AccessKey,
		SecretKey:    creds.SecretKey,
		SessionToken: creds.SessionToken,
	})

	return nil
}
