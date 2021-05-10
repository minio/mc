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
