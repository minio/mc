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
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminUserSvcAcctDisableCmd = cli.Command{
	Name:         "disable",
	Usage:        "Disable a services account",
	Action:       mainAdminUserSvcAcctDisable,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS SERVICE-ACCOUNT

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Disable the service account 'J123C4ZXEQN8RK6ND35I' in MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ J123C4ZXEQN8RK6ND35I
`,
}

// checkAdminUserSvcAcctDisableSyntax - validate all the passed arguments
func checkAdminUserSvcAcctDisableSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user svcacct disable command.")
	}
}

// mainAdminUserSvcAcctDisable is the handle for "mc admin user svcacct disable" command.
func mainAdminUserSvcAcctDisable(ctx *cli.Context) error {
	checkAdminUserSvcAcctDisableSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	svcAccount := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	opts := madmin.UpdateServiceAccountReq{
		NewStatus: "off",
	}

	e := client.UpdateServiceAccount(globalContext, svcAccount, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get disable the specified service account")

	printMsg(svcAcctMessage{
		op:        "disable",
		AccessKey: svcAccount,
	})

	return nil
}
