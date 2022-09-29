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
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminGroupPolicyDetachCmd = cli.Command{
	Name:         "detach",
	Usage:        "detach an IAM policy from a group",
	Action:       mainAdminGroupPolicyDetach,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME GROUPNAME

POLICYNAME:
  Name of the policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Unset the "diagnostics" policy for user "james".
     {{.Prompt}} {{.HelpName}} myminio diagnostics user=james

  2. Set the "diagnostics" policy for group "auditors".
     {{.Prompt}} {{.HelpName}} myminio diagnostics group=auditors
`,
}

func checkAdminGroupPolicyDetachSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		showCommandHelpAndExit(ctx, "detach", 1) // last argument is exit code
	}
}

func detachCannedPolicies(existingPolicies, policiesToRemove string) (string, error) {
	policiesToRemove = strings.TrimSpace(policiesToRemove)
	if policiesToRemove == "" {
		return "", errors.New("empty policy name is not supported")
	}
	filteredPolicies := strings.Split(existingPolicies, ",")
	for _, p1 := range strings.Split(policiesToRemove, ",") {
		found := false
		for i, p2 := range filteredPolicies {
			if p1 == p2 {
				found = true
				filteredPolicies = append(filteredPolicies[:i], filteredPolicies[i+1:]...)
				break
			}
		}
		if !found {
			return "", fmt.Errorf("policy `%s` not found", p1)
		}

	}
	return strings.Join(filteredPolicies, ","), nil
}

// mainAdminUserPolicyDetach is the handler for "mc admin group policy detach" command.
func mainAdminGroupPolicyDetach(ctx *cli.Context) error {
	checkAdminGroupPolicyDetachSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policiesToDetach := args.Get(1)
	group := args.Get(2)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var existingPolicies string

	groupInfo, e := client.GetGroupDescription(globalContext, group)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get group policy info")
	existingPolicies = groupInfo.Policy

	newPolicies, e := removeCannedPolicies(existingPolicies, policiesToDetach)
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to detach the policy")
	}

	e = client.SetPolicy(globalContext, newPolicies, group, true)
	if e == nil {
		printMsg(userPolicyMessage{
			op:          "detach",
			Policy:      policiesToDetach,
			UserOrGroup: group,
			IsGroup:     true,
		})
	} else {
		fatalIf(probe.NewError(e), "Unable to detach the policy")
	}
	return nil
}
