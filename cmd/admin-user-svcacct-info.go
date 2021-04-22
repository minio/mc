/*
 * MinIO Client (C) 2021 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var adminUserSvcAcctInfoFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "policy",
		Usage: "print policy is JSON format",
	},
}

var adminUserSvcAcctInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "Get a service account info",
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
  1. Get information of service account 'J123C4ZXEQN8RK6ND35I'
     {{.Prompt}} {{.HelpName}} myminio/ J123C4ZXEQN8RK6ND35I
`,
}

// checkAdminUserSvcAcctInfoSyntax - validate all the passed arguments
func checkAdminUserSvcAcctInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user svcacct info command.")
	}
}

// mainAdminUserSvcAcctInfo is the handle for "mc admin user svcacct info" command.
func mainAdminUserSvcAcctInfo(ctx *cli.Context) error {
	checkAdminUserSvcAcctInfoSyntax(ctx)

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
		fmt.Println(svcInfo.Policy)
		return nil
	}

	printMsg(svcAcctMessage{
		op:            "info",
		AccessKey:     svcAccount,
		AccountStatus: svcInfo.AccountStatus,
		ParentUser:    svcInfo.ParentUser,
		ImpliedPolicy: svcInfo.ImpliedPolicy,
		Policy:        svcInfo.Policy,
	})

	return nil
}
