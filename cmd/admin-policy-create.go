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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminPolicyCreateCmd = cli.Command{
	Name:         "create",
	Usage:        "create a new IAM policy",
	Action:       mainAdminPolicyCreate,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME POLICYFILE

POLICYNAME:
  Name of the canned policy on MinIO server.

POLICYFILE:
  Name of the policy file associated with the policy name.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a new canned policy 'writeonly'.
     {{.Prompt}} {{.HelpName}} myminio writeonly /tmp/writeonly.json
 `,
}

// checkAdminPolicyCreateSyntax - validate all the passed arguments
func checkAdminPolicyCreateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// userPolicyMessage container for content message structure
type userPolicyMessage struct {
	op          string
	Status      string            `json:"status"`
	Policy      string            `json:"policy,omitempty"`
	PolicyInfo  madmin.PolicyInfo `json:"policyInfo,omitempty"`
	UserOrGroup string            `json:"userOrGroup,omitempty"`
	IsGroup     bool              `json:"isGroup"`
}

func (u userPolicyMessage) accountType() string {
	switch u.op {
	case "attach", "detach":
		if u.IsGroup {
			return "group"
		}
		return "user"
	}
	return ""
}

func (u userPolicyMessage) String() string {
	switch u.op {
	case "info":
		buf, e := json.MarshalIndent(u.PolicyInfo, "", " ")
		fatalIf(probe.NewError(e), "Unable to marshal to JSON.")
		return string(buf)
	case "list":
		return console.Colorize("PolicyName", u.Policy)
	case "remove":
		return console.Colorize("PolicyMessage", "Removed policy `"+u.Policy+"` successfully.")
	case "create":
		return console.Colorize("PolicyMessage", "Created policy `"+u.Policy+"` successfully.")
	case "detach":
		return console.Colorize("PolicyMessage",
			fmt.Sprintf("Policy `%s` successfully detached from %s `%s`", u.Policy, u.accountType(), u.UserOrGroup))
	case "attach":
		return console.Colorize("PolicyMessage",
			fmt.Sprintf("Policy `%s` successfully attached to %s `%s`", u.Policy, u.accountType(), u.UserOrGroup))
	}

	return ""
}

func (u userPolicyMessage) JSON() string {
	u.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// mainAdminPolicyCreate is the handle for "mc admin policy create" command.
func mainAdminPolicyCreate(ctx *cli.Context) error {
	checkAdminPolicyCreateSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	policy, e := os.ReadFile(args.Get(2))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get policy")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	fatalIf(probe.NewError(client.AddCannedPolicy(globalContext, args.Get(1), policy)).Trace(args...), "Unable to create new policy")

	printMsg(userPolicyMessage{
		op:     ctx.Command.Name,
		Policy: args.Get(1),
	})

	return nil
}
