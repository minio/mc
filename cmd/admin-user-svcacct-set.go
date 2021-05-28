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
	"io/ioutil"

	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminUserSvcAcctSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "secret-key",
		Usage: "set a secret key for the service account",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "path to a JSON policy file",
	},
}

var adminUserSvcAcctSetCmd = cli.Command{
	Name:         "set",
	Usage:        "edit an existing service account",
	Action:       mainAdminUserSvcAcctSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUserSvcAcctSetFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS SERVICE-ACCOUNT

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Change the secret key of the service account 'J123C4ZXEQN8RK6ND35I' in MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ 'J123C4ZXEQN8RK6ND35I' --secret-key 'xxxxxxx'
`,
}

// checkAdminUserSvcAcctSetSyntax - validate all the passed arguments
func checkAdminUserSvcAcctSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user svcacct set command.")
	}
}

// mainAdminUserSvcAcctSet is the handle for "mc admin user svcacct set" command.
func mainAdminUserSvcAcctSet(ctx *cli.Context) error {
	checkAdminUserSvcAcctSetSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	svcAccount := args.Get(1)

	secretKey := ctx.String("secret-key")
	policyPath := ctx.String("policy")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var buf []byte
	if policyPath != "" {
		var e error
		buf, e = ioutil.ReadFile(policyPath)
		fatalIf(probe.NewError(e), "Unable to open the policy document.")
	}

	opts := madmin.UpdateServiceAccountReq{
		NewPolicy:    buf,
		NewSecretKey: secretKey,
	}

	e := client.UpdateServiceAccount(globalContext, svcAccount, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add a new service account")

	printMsg(svcAcctMessage{
		op:        "set",
		AccessKey: svcAccount,
	})

	return nil
}
