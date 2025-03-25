// Copyright (c) 2015-2024 MinIO, Inc.
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
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/policy"
)

var idpLdapAccesskeyCreateFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "access-key",
		Usage: "set an access key for the account",
	},
	cli.StringFlag{
		Name:  "secret-key",
		Usage: "set a secret key for the  account",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "path to a JSON policy file",
	},
	cli.StringFlag{
		Name:  "name",
		Usage: "friendly name for the account",
	},
	cli.StringFlag{
		Name:  "description",
		Usage: "description for the account",
	},
	cli.StringFlag{
		Name:  "expiry-duration",
		Usage: "duration before the access key expires",
	},
	cli.StringFlag{
		Name:  "expiry",
		Usage: "expiry date for the access key",
	},
	cli.BoolFlag{
		Name:   "login",
		Usage:  "log in using ldap credentials to generate access key pair for future use",
		Hidden: true,
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

  2. Create a new access key pair with custom access key and secret key
     {{.Prompt}} {{.HelpName}} local/ --access-key myaccesskey --secret-key mysecretkey

  4. Create a new access key pair for user with username "james" that expires in 1 day
     {{.Prompt}} {{.HelpName}} local/ james --expiry-duration 24h

  5. Create a new access key pair for authenticated user that expires on 2021-01-01
     {{.Prompt}} {{.HelpName}} --expiry 2021-01-01
`,
}

func mainIDPLdapAccesskeyCreate(ctx *cli.Context) error {
	return commonAccesskeyCreate(ctx, true)
}

func commonAccesskeyCreate(ctx *cli.Context, ldap bool) error {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	targetUser := args.Get(1)

	if ctx.Bool("login") {
		deprecatedError("mc idp ldap accesskey create-with-login")
	}

	opts := accessKeyCreateOpts(ctx, targetUser)
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var res madmin.Credentials
	var e error
	if ldap {
		res, e = client.AddServiceAccountLDAP(globalContext, opts)
	} else {
		res, e = client.AddServiceAccount(globalContext, opts)
	}
	fatalIf(probe.NewError(e), "Unable to add service account.")

	m := accesskeyMessage{
		op:          "create",
		Status:      "success",
		AccessKey:   res.AccessKey,
		SecretKey:   res.SecretKey,
		Expiration:  &res.Expiration,
		Name:        opts.Name,
		Description: opts.Description,
	}
	printMsg(m)

	return nil
}

func accessKeyCreateOpts(ctx *cli.Context, targetUser string) madmin.AddServiceAccountReq {
	name := ctx.String("name")
	expVal := ctx.String("expiry")
	policyPath := ctx.String("policy")
	accessKey := ctx.String("access-key")
	secretKey := ctx.String("secret-key")
	description := ctx.String("description")
	expDurVal := ctx.Duration("expiry-duration")

	// generate access key and secret key
	if len(accessKey) <= 0 || len(secretKey) <= 0 {
		randomAccessKey, randomSecretKey, err := generateCredentials()
		if err != nil {
			fatalIf(err, "unable to generate randomized access credentials")
		}
		if len(accessKey) <= 0 {
			accessKey = randomAccessKey
		}
		if len(secretKey) <= 0 {
			secretKey = randomSecretKey
		}
	}

	if expVal != "" && expDurVal != 0 {
		fatalIf(probe.NewError(errors.New("Only one of --expiry or --expiry-duration can be specified")), "invalid flags")
	}

	opts := madmin.AddServiceAccountReq{
		TargetUser:  targetUser,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		Name:        name,
		Description: description,
	}

	if policyPath != "" {
		// Validate the policy document and ensure it has at least one statement
		policyBytes, e := os.ReadFile(policyPath)
		fatalIf(probe.NewError(e), "unable to read the policy document")

		p, e := policy.ParseConfig(bytes.NewReader(policyBytes))
		fatalIf(probe.NewError(e), "unable to parse the policy document")

		if p.IsEmpty() {
			fatalIf(errInvalidArgument(), "empty policies are not allowed")
		}

		opts.Policy = policyBytes
	}

	switch {
	case expVal != "":
		location, e := time.LoadLocation("Local")
		fatalIf(probe.NewError(e), "unable to load local location. verify your local TZ=<val> settings")

		var found bool
		for _, format := range supportedTimeFormats {
			t, e := time.ParseInLocation(format, expVal, location)
			if e == nil {
				found = true
				opts.Expiration = &t
				break
			}
		}

		if !found {
			fatalIf(probe.NewError(fmt.Errorf("invalid expiry date format '%s'", expVal)), "unable to parse the expiry argument")
		}
	case expDurVal != 0:
		t := time.Now().Add(expDurVal)
		opts.Expiration = &t
	}

	return opts
}
