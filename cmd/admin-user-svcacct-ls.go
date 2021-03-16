/*
 * MinIO Client (C) 2021 MinIO, Inc.
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

var adminUserSvcAcctListCmd = cli.Command{
	Name:         "ls",
	Usage:        "List services accounts",
	Action:       mainAdminUserSvcAcctList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS TARGET-ACCOUNT

TARGET-ACCOUNT:
  Could be a MinIO user, STS or LDAP account.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all service accounts for user 'foobar'.
     {{.Prompt}} {{.HelpName}} myminio/ foobar
`,
}

// checkAdminUserSvcAcctListSyntax - validate all the passed arguments
func checkAdminUserSvcAcctListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user svcacct ls command.")
	}
}

// mainAdminUserSvcAcctList is the handle for "mc admin user svcacct ls" command.
func mainAdminUserSvcAcctList(ctx *cli.Context) error {
	checkAdminUserSvcAcctListSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	svcList, e := client.ListServiceAccounts(globalContext, user)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add a new service account")

	for _, svc := range svcList.Accounts {
		printMsg(svcAcctMessage{
			op:        "ls",
			AccessKey: svc,
		})
	}

	return nil
}
