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
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
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
	cli.StringFlag{
		Name:  "name",
		Usage: "name for the service account",
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

var adminUserSvcAcctSetCmd = cli.Command{
	Name:         "edit",
	Aliases:      []string{"set"},
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

  2. Change the expiry of the service account 'J123C4ZXEQN8RK6ND35I' in MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ 'J123C4ZXEQN8RK6ND35I' --expiry 2023-06-24T10:00:00-07:00
`,
}

// checkAdminUserSvcAcctSetSyntax - validate all the passed arguments
func checkAdminUserSvcAcctSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1)
	}
}

// mainAdminUserSvcAcctSet is the handle for "mc admin user svcacct set" command.
func mainAdminUserSvcAcctSet(ctx *cli.Context) error {
	checkAdminUserSvcAcctSetSyntax(ctx)

	console.SetColor("AccMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	svcAccount := args.Get(1)

	secretKey := ctx.String("secret-key")
	policyPath := ctx.String("policy")
	name := ctx.String("name")
	description := ctx.String("description")
	expiry := ctx.String("expiry")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var buf []byte
	if policyPath != "" {
		var e error
		buf, e = os.ReadFile(policyPath)
		fatalIf(probe.NewError(e), "Unable to open the policy document.")
	}

	var expiryTime time.Time
	var expiryPointer *time.Time

	if expiry != "" {
		location, e := time.LoadLocation("Local")
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to parse the expiry argument.")
		}

		patternMatched := false
		for _, format := range supportedTimeFormats {
			t, e := time.ParseInLocation(format, expiry, location)
			if e == nil {
				patternMatched = true
				expiryTime = t
				expiryPointer = &expiryTime
				break
			}
		}

		if !patternMatched {
			fatalIf(probe.NewError(fmt.Errorf("expiry argument is not matching any of the supported patterns")), "unable to parse the expiry argument.")
		}
	}

	opts := madmin.UpdateServiceAccountReq{
		NewPolicy:      buf,
		NewSecretKey:   secretKey,
		NewName:        name,
		NewDescription: description,
		NewExpiration:  expiryPointer,
	}

	e := client.UpdateServiceAccount(globalContext, svcAccount, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to edit the specified service account")

	printMsg(acctMessage{
		op:        svcAccOpSet,
		AccessKey: svcAccount,
	})

	return nil
}
