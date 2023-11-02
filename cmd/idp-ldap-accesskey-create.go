// Copyright (c) 2015-2023 MinIO, Inc.
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
	"time"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var idpLdapAccesskeyCreateFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "access-key",
		Usage: "set an access key for the service account",
	},
	cli.StringFlag{
		Name:  "secret-key",
		Usage: "set a secret key for the service account",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "path to a JSON policy file",
	},
	cli.StringFlag{
		Name:  "name",
		Usage: "friendly name for the service account",
	},
	cli.StringFlag{
		Name:  "description",
		Usage: "description for the service account",
	},
	cli.StringFlag{
		Name:  "expiry",
		Usage: "time of expiration for the service account",
	},
}

var idpLdapAccesskeyCreateCmd = cli.Command{
	Name:         "create",
	Usage:        "create access key pairs for LDAP",
	Action:       mainIDPLdapAccesskeyCreate,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapAccesskeyCreateFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a new access key pair with the same policy as the authenticated user
     {{.Prompt}} {{.HelpName}} local/
	`,
}

func mainIDPLdapAccesskeyCreate(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	targetUser := args.Get(1)

	expVal := ctx.Duration("expiry")
	exp := time.Now().Add(expVal)
	accessVal := ctx.String("accesskey")
	secretVal := ctx.String("secretkey")

	if expVal == 0 {
		exp = time.Unix(0, 0)
	}

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	accessKey, secretKey, e := generateCredentials()
	fatalIf(probe.NewError(e), "Unable to generate credentials.")

	// If access key and secret key are provided, use them instead
	if accessVal != "" {
		accessKey = accessVal
	}
	if secretVal != "" {
		secretKey = secretVal
	}

	res, e := client.AddServiceAccountLDAP(globalContext,
		madmin.AddServiceAccountReq{
			TargetUser: targetUser,
			AccessKey:  accessKey,
			SecretKey:  secretKey,
			Expiration: &exp,
		})
	fatalIf(probe.NewError(e), "Unable to add service account.")

	m := ldapAccesskeyMessage{
		op:         "create",
		Status:     "success",
		AccessKey:  res.AccessKey,
		SecretKey:  res.SecretKey,
		Expiration: &res.Expiration,
	}
	printMsg(m)

	return nil
}
