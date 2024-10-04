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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
	"github.com/minio/pkg/v3/policy"
)

var adminUserPolicyCmd = cli.Command{
	Name:         "policy",
	Usage:        "export user policies in JSON format",
	Action:       mainAdminUserPolicy,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
USAGE:
  {{.HelpName}} TARGET USERNAME
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display the policy document of a user "foobar" in JSON format.
     {{.Prompt}} {{.HelpName}} myminio foobar
`,
}

// checkAdminUserPolicySyntax - validate all the passed arguments
func checkAdminUserPolicySyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainAdminUserPolicy is the handler for "mc admin user policy" command.
func mainAdminUserPolicy(ctx *cli.Context) error {
	checkAdminUserPolicySyntax(ctx)

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	user, e := client.GetUserInfo(globalContext, args.Get(1))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get user info")

	if user.PolicyName == "" {
		e = fmt.Errorf("policy not found for user %s", args.Get(1))
		fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch user policy document")
	}

	policyNames := strings.Split(user.PolicyName, ",")

	var policies []policy.Policy
	for _, policyName := range policyNames {
		if policyName == "" {
			continue
		}
		policyInfo, e := getPolicyInfo(client, policyName)
		fatalIf(probe.NewError(e).Trace(), "Unable to fetch user policy document for policy "+policyName)

		var policyObj policy.Policy
		if e := json.Unmarshal(policyInfo.Policy, &policyObj); e != nil {
			fatalIf(probe.NewError(e).Trace(), "Unable to unmarshal policy")
		}
		policies = append(policies, policyObj)
	}

	mergedPolicy := policy.MergePolicies(policies...)
	json.NewEncoder(os.Stdout).Encode(mergedPolicy)
	return nil
}
