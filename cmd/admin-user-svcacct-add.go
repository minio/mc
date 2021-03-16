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
	"os"
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	iampolicy "github.com/minio/minio/pkg/iam/policy"
	"github.com/minio/minio/pkg/madmin"
)

var adminUserSvcAcctAddFlags = []cli.Flag{
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
}

var adminUserSvcAcctAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a new service account",
	Action:       mainAdminUserSvcAcctAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUserSvcAcctAddFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS ACCOUNT

ACCOUNT:
  An account could be a regular MinIO user, STS ou LDAP user.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add a new service account for user 'foobar' to MinIO server.
     {{.Prompt}} {{.HelpName}} myminio foobar
`,
}

// checkAdminUserSvcAcctAddSyntax - validate all the passed arguments
func checkAdminUserSvcAcctAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user svcacct add command.")
	}
}

// svcAcctMessage container for content message structure
type svcAcctMessage struct {
	op            string
	Status        string   `json:"status"`
	AccessKey     string   `json:"accessKey,omitempty"`
	SecretKey     string   `json:"secretKey,omitempty"`
	ParentUser    string   `json:"parentUser,omitempty"`
	ImpliedPolicy bool     `json:"impliedPolicy,omitempty"`
	Policy        string   `json:"policy,omitempty"`
	AccountStatus string   `json:"accountStatus,omitempty"`
	MemberOf      []string `json:"memberOf,omitempty"`
}

func (u svcAcctMessage) String() string {
	switch u.op {
	case "ls":
		accessFieldMaxLen := 20
		/*
			userFieldMaxLen := 9
			policyFieldMaxLen := 20
		*/

		// Create a new pretty table with cols configuration
		return newPrettyTable("  ",
			Field{"AccessKey", accessFieldMaxLen},
		).buildRow(u.AccessKey)
	case "info":
		policyField := ""
		if u.ImpliedPolicy {
			policyField = "implied"
		} else {
			policyField = "embedded"
		}
		return console.Colorize("UserMessage", strings.Join(
			[]string{
				fmt.Sprintf("AccessKey: %s", u.AccessKey),
				fmt.Sprintf("ParentUser: %s", u.ParentUser),
				fmt.Sprintf("Status: %s", u.AccountStatus),
				fmt.Sprintf("Policy: %s", policyField),
				// fmt.Sprintf("MemberOf: %s", strings.Join(u.MemberOf, ",")),
			}, "\n"))
	case "rm":
		return console.Colorize("UserMessage", "Removed service account `"+u.AccessKey+"` successfully.")
	case "disable":
		return console.Colorize("UserMessage", "Disabled service account `"+u.AccessKey+"` successfully.")
	case "enable":
		return console.Colorize("UserMessage", "Enabled service account `"+u.AccessKey+"` successfully.")
	case "add":
		return console.Colorize("UserMessage", "Added service account `"+u.AccessKey+"` successfully.")
	case "set":
		return console.Colorize("UserMessage", "Edited service account `"+u.AccessKey+"` successfully.")
	}
	return ""
}

func (u svcAcctMessage) JSON() string {
	u.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// mainAdminUserSvcAcctAdd is the handle for "mc admin user svcacct add" command.
func mainAdminUserSvcAcctAdd(ctx *cli.Context) error {
	checkAdminUserSvcAcctAddSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1)

	accessKey := ctx.String("access-key")
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

	opts := madmin.AddServiceAccountReq{
		Policy:     policy,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		TargetUser: user,
	}

	creds, e := client.AddServiceAccount(globalContext, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add a new service account")

	printMsg(svcAcctMessage{
		op:            "add",
		AccessKey:     creds.AccessKey,
		SecretKey:     creds.SecretKey,
		AccountStatus: "enabled",
	})

	return nil
}
