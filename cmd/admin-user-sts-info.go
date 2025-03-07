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

var adminUserSTSAcctSubcommands = []cli.Command{
	adminUserSTSAcctInfoCmd,
}

var adminUserSTSAcctCmd = cli.Command{
	Name:            "sts",
	Usage:           "manage STS accounts",
	Action:          mainAdminUserSTSAcct,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     adminUserSTSAcctSubcommands,
	HideHelpCommand: true,
}

// mainAdminUserSTSAcct is the handle for "mc admin user sts" command.
func mainAdminUserSTSAcct(ctx *cli.Context) error {
	commandNotFound(ctx, adminUserSTSAcctSubcommands)
	return nil
}

var adminUserSTSAcctInfoFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "policy",
		Usage: "print policy in JSON format",
	},
}

var adminUserSTSAcctInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display temporary account info",
	Action:       mainAdminUserSTSAcctInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUserSTSAcctInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS STS-ACCOUNT

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display information for the temporary account 'J123C4ZXEQN8RK6ND35I'
     {{.Prompt}} {{.HelpName}} myminio/ J123C4ZXEQN8RK6ND35I
`,
}

// checkAdminUserSTSAcctInfoSyntax - validate all the passed arguments
func checkAdminUserSTSAcctInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1)
	}
}

// mainAdminUserSTSAcctInfo is the handle for "mc admin user sts info" command.
func mainAdminUserSTSAcctInfo(ctx *cli.Context) error {
	checkAdminUserSTSAcctInfoSyntax(ctx)

	console.SetColor("AccountMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	stsAccount := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	stsInfo, e := client.TemporaryAccountInfo(globalContext, stsAccount)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get information of the specified service account")

	if ctx.Bool("policy") {
		if stsInfo.Policy == "" {
			fatalIf(errDummy().Trace(args...), "No policy found associated to the specified service account. Check the policy of its parent user.")
		}
		p, e := policy.ParseConfig(strings.NewReader(stsInfo.Policy))
		fatalIf(probe.NewError(e).Trace(args...), "Unable to parse policy.")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		fatalIf(probe.NewError(enc.Encode(p)).Trace(args...), "Unable to write policy to stdout.")
		return nil
	}

	printMsg(acctMessage{
		op:            svcAccOpInfo,
		AccessKey:     stsAccount,
		AccountStatus: stsInfo.AccountStatus,
		ParentUser:    stsInfo.ParentUser,
		ImpliedPolicy: stsInfo.ImpliedPolicy,
		Policy:        json.RawMessage(stsInfo.Policy),
		Expiration:    stsInfo.Expiration,
	})

	return nil
}
