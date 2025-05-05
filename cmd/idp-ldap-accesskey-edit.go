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

var idpLdapAccesskeyEditFlags = []cli.Flag{
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
}

var idpLdapAccesskeyEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "edit existing access keys for LDAP",
	Action:       mainIDPLdapAccesskeyEdit,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapAccesskeyEditFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Change the secret key for the access key "testkey"
     {{.Prompt}} {{.HelpName}} myminio/ testkey --secret-key 'xxxxxxx'
  2. Change the expiry duration for the access key "testkey"
     {{.Prompt}} {{.HelpName}} myminio/ testkey ---expiry-duration 24h
`,
}

func mainIDPLdapAccesskeyEdit(ctx *cli.Context) error {
	return commonAccesskeyEdit(ctx)
}

func commonAccesskeyEdit(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	accessKey := args.Get(1)

	opts := accessKeyEditOpts(ctx)
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.UpdateServiceAccount(globalContext, accessKey, opts)
	fatalIf(probe.NewError(e), "Unable to edit service account.")

	m := accesskeyMessage{
		op:        "edit",
		Status:    "success",
		AccessKey: accessKey,
	}
	printMsg(m)

	return nil
}

func accessKeyEditOpts(ctx *cli.Context) madmin.UpdateServiceAccountReq {
	name := ctx.String("name")
	expVal := ctx.String("expiry")
	policyPath := ctx.String("policy")
	secretKey := ctx.String("secret-key")
	description := ctx.String("description")
	expDurVal := ctx.Duration("expiry-duration")

	if name == "" && expVal == "" && expDurVal == 0 && policyPath == "" && secretKey == "" && description == "" {
		fatalIf(probe.NewError(errors.New("At least one property must be edited")), "invalid flags")
	}

	if expVal != "" && expDurVal != 0 {
		fatalIf(probe.NewError(errors.New("Only one of --expiry or --expiry-duration can be specified")), "invalid flags")
	}

	opts := madmin.UpdateServiceAccountReq{
		NewName:        name,
		NewSecretKey:   secretKey,
		NewDescription: description,
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

		opts.NewPolicy = policyBytes
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
				opts.NewExpiration = &t
				break
			}
		}

		if !found {
			fatalIf(probe.NewError(fmt.Errorf("invalid expiry date format '%s'", expVal)), "unable to parse the expiry argument")
		}
	case expDurVal != 0:
		t := time.Now().Add(expDurVal)
		opts.NewExpiration = &t
	}

	return opts
}
