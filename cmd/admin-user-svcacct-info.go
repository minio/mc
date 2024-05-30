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
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
	"github.com/minio/pkg/v3/policy"
)

var adminUserSvcAcctInfoFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "policy",
		Usage: "print policy in JSON format",
	},
}

var adminUserSvcAcctInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display service account info",
	Action:       mainAdminUserSvcAcctInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUserSvcAcctInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS SERVICE-ACCOUNT

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display information for service account 'J123C4ZXEQN8RK6ND35I'
     {{.Prompt}} {{.HelpName}} myminio/ J123C4ZXEQN8RK6ND35I
`,
}

// checkAdminUserSvcAcctInfoSyntax - validate all the passed arguments
func checkAdminUserSvcAcctInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1)
	}
}

// mainAdminUserSvcAcctInfo is the handle for "mc admin user svcacct info" command.
func mainAdminUserSvcAcctInfo(ctx *cli.Context) error {
	checkAdminUserSvcAcctInfoSyntax(ctx)

	console.SetColor("AccMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	svcAccount := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	svcInfo, e := client.InfoServiceAccount(globalContext, svcAccount)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get information of the specified service account")

	if ctx.Bool("policy") {
		if svcInfo.Policy == "" {
			fatalIf(errDummy().Trace(args...), "No policy found associated to the specified service account. Check the policy of its parent user.")
		}
		p, e := policy.ParseConfig(strings.NewReader(svcInfo.Policy))
		fatalIf(probe.NewError(e).Trace(args...), "Unable to parse policy.")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		fatalIf(probe.NewError(enc.Encode(p)).Trace(args...), "Unable to write policy to stdout.")
		return nil
	}

	printMsg(acctMessage{
		op:            svcAccOpInfo,
		AccessKey:     svcAccount,
		Name:          svcInfo.Name,
		Description:   svcInfo.Description,
		AccountStatus: svcInfo.AccountStatus,
		ParentUser:    svcInfo.ParentUser,
		ImpliedPolicy: svcInfo.ImpliedPolicy,
		Policy:        json.RawMessage(svcInfo.Policy),
		Expiration:    svcInfo.Expiration,
	})

	return nil
}
