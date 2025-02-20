// Copyright (c) 2015-2022 MinIO, Inc.
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

var adminUserSTSAcctRevokeFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "all",
		Usage: "revoke all STS accounts for the specified user",
	},
	cli.StringFlag{
		Name:  "token-type",
		Usage: "specify the token type to revoke",
	},
}

var adminUserSTSAcctRevokeCmd = cli.Command{
	Name:         "revoke",
	Usage:        "revokes all STS accounts or specified types for the specified user",
	Action:       mainAdminUserSTSAcctRevoke,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUserSTSAcctRevokeFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS USER [--all | --token-type TOKEN_TYPE]

  Exactly one of --all or --token-type must be specified.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display information for the temporary account 'J123C4ZXEQN8RK6ND35I'
     {{.Prompt}} {{.HelpName}} myminio/ J123C4ZXEQN8RK6ND35I
`,
}

// checkAdminUserSTSAcctInfoSyntax - validate all the passed arguments
func checkAdminUserSTSAcctRevokeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1)
	}

	// All flag is here to ensure that the user wants to revoke all tokens.
	// It is not actually sent, since an empty token type is sent to revoke all tokens.
	if !ctx.Bool("all") && ctx.String("token-type") == "" {
		fatalIf(errDummy().Trace(), "Exactly one of --all or --token-type must be specified.")
	}
}

// mainAdminUserSTSAcctInfo is the handle for "mc admin user sts info" command.
func mainAdminUserSTSAcctRevoke(ctx *cli.Context) error {
	checkAdminUserSTSAcctInfoSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1)
	tokenType := ctx.String("token-type")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.RevokeTokens(globalContext, user, tokenType)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to revoke tokens for %s", user)

	printMsg(userMessage{
		op:        ctx.Command.Name,
		AccessKey: args.Get(1),
	})

	return nil
}
