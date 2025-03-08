// Copyright (c) 2015-2025 MinIO, Inc.
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
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var idpLdapAccesskeySTSRevokeCmd = cli.Command{
	Name:         "sts-revoke",
	Usage:        "revokes all STS accounts or specified types for the specified user",
	Action:       mainIdpLdapAccesskeySTSRevoke,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminAccesskeySTSRevokeFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS USER [--all | --token-type TOKEN_TYPE]

  Exactly one of --all or --token-type must be specified.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Revoke all 
	 {{.Prompt}} {{.HelpName}} myminio user1 --all
  2. TODO
	 {{.Prompt}} {{.HelpName}} myminio user1 --token-type app-1
`,
}

// mainIdpLdapUserSTSAcctInfo is the handle for "mc admin accesskey sts-revoke" command.
func mainIdpLdapAccesskeySTSRevoke(ctx *cli.Context) error {
	checkAdminAccesskeySTSRevokeSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1) // will be empty if --self flag is set
	tokenRevokeType := ctx.String("token-type")
	fullRevoke := ctx.Bool("all")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.RevokeTokens(globalContext, madmin.RevokeTokensReq{
		User:            user,
		TokenRevokeType: tokenRevokeType,
		FullRevoke:      fullRevoke,
	})
	fatalIf(probe.NewError(e).Trace(args...), "Unable to revoke tokens for %s", user)

	printMsg(stsRevokeMessage{
		User:            user,
		TokenRevokeType: tokenRevokeType,
	})

	return nil
}
