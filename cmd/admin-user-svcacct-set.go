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
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	iampolicy "github.com/minio/minio/pkg/iam/policy"
	"github.com/minio/minio/pkg/madmin"
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
}

var adminUserSvcAcctSetCmd = cli.Command{
	Name:         "set",
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
`,
}

// checkAdminUserSvcAcctSetSyntax - validate all the passed arguments
func checkAdminUserSvcAcctSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user svcacct set command.")
	}
}

// mainAdminUserSvcAcctSet is the handle for "mc admin user svcacct set" command.
func mainAdminUserSvcAcctSet(ctx *cli.Context) error {
	checkAdminUserSvcAcctSetSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	svcAccount := args.Get(1)

	secretKey := ctx.String("secret-key")
	policyPath := ctx.String("policy")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var policy *iampolicy.Policy
	if policyPath != "" {
		var e error
		f, e := os.Open(policyPath)
		fatalIf(probe.NewError(e), "Unable to open the policy document.")
		policy, e = iampolicy.ParseConfig(f)
		fatalIf(probe.NewError(e), "Unable to parse the policy document.")
	}

	opts := madmin.UpdateServiceAccountReq{
		NewPolicy:    policy,
		NewSecretKey: secretKey,
	}

	e := client.UpdateServiceAccount(globalContext, svcAccount, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add a new service account")

	printMsg(svcAcctMessage{
		op:        "set",
		AccessKey: svcAccount,
	})

	return nil
}
