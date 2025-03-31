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
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var adminAccesskeySTSRevokeFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "all",
		Usage: "revoke all STS accounts for the specified user",
	},
	cli.BoolFlag{
		Name:  "self",
		Usage: "revoke all STS accounts for the authenticated user",
	},
	cli.StringFlag{
		Name:  "token-type",
		Usage: "specify the token type to revoke",
	},
}

var adminAccesskeySTSRevokeCmd = cli.Command{
	Name:         "sts-revoke",
	Usage:        "revokes all STS accounts or specified types for the specified user",
	Action:       mainAdminAccesskeySTSRevoke,
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
  1. Revoke all STS accounts for user "user1"
	 {{.Prompt}} {{.HelpName}} myminio user1 --all

  2. Revoke STS accounts of a token type "app-1" for user "user1"
	 {{.Prompt}} {{.HelpName}} myminio user1 --token-type app-1

  3. Revoke all STS accounts for the authenticated user
	 {{.Prompt}} {{.HelpName}} myminio --self

  4. Revoke STS accounts of a token type "app-1" for the authenticated user
	 {{.Prompt}} {{.HelpName}} myminio --self --token-type app-1
`,
}

type stsRevokeMessage struct {
	Status          string `json:"status"`
	User            string `json:"user"`
	TokenRevokeType string `json:"tokenRevokeType,omitempty"`
}

func (m stsRevokeMessage) String() string {
	userString := "user " + m.User
	if m.User == "" {
		userString = "authenticated user"
	}
	if m.TokenRevokeType == "" {
		return "Successfully revoked all STS accounts for " + userString
	}
	return "Successfully revoked all STS accounts of type " + m.TokenRevokeType + " for " + userString
}

func (m stsRevokeMessage) JSON() string {
	if m.Status == "" {
		m.Status = "success"
	}
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// checkSTSRevokeSyntax - validate all the passed arguments
func checkSTSRevokeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 2 || len(ctx.Args()) == 0 {
		showCommandHelpAndExit(ctx, 1)
	}

	if !ctx.Bool("self") && ctx.Args().Get(1) == "" {
		fatalIf(errInvalidArgument().Trace(), "Must specify user or use --self flag.")
	}

	if ctx.Bool("self") && ctx.Args().Get(1) != "" {
		fatalIf(errInvalidArgument().Trace(), "Cannot specify user with --self flag.")
	}

	if (!ctx.Bool("all") && ctx.String("token-type") == "") || (ctx.Bool("all") && ctx.String("token-type") != "") {
		fatalIf(errDummy().Trace(), "Exactly one of --all or --token-type must be specified.")
	}
}

// mainAdminAccesskeySTSRevoke is the handle for "mc admin accesskey sts-revoke" command.
func mainAdminAccesskeySTSRevoke(ctx *cli.Context) error {
	checkSTSRevokeSyntax(ctx)

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
