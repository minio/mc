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
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminPolicyUnsetCmd = cli.Command{
	Name:         "unset",
	Usage:        "unset IAM policy from a user or group",
	Action:       mainAdminPolicyUnset,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME [ user=username1 | group=groupname1 ]

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

func checkAdminPolicyUnsetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, "unset", 1) // last argument is exit code
	}
}

func removeCannedPolicies(existingPolicies, policiesToRemove string) (string, error) {
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

// mainAdminPolicyUnset is the handler for "mc admin policy unset" command.
func mainAdminPolicyUnset(ctx *cli.Context) error {
	checkAdminPolicyUnsetSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policiesToUnset := args.Get(1)
	entityArg := args.Get(2)

	userOrGroup, isGroup, e1 := parseEntityArg(entityArg)
	fatalIf(probe.NewError(e1).Trace(args...), "Bad last argument")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var existingPolicies string

	if !isGroup {
		userInfo, e := client.GetUserInfo(globalContext, userOrGroup)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get user policy info")
		existingPolicies = userInfo.PolicyName
	} else {
		groupInfo, e := client.GetGroupDescription(globalContext, userOrGroup)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get group policy info")
		existingPolicies = groupInfo.Policy
	}

	newPolicies, e := removeCannedPolicies(existingPolicies, policiesToUnset)
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to unset the policy")
	}

	e = client.SetPolicy(globalContext, newPolicies, userOrGroup, isGroup)
	if e == nil {
		printMsg(userPolicyMessage{
			op:          "unset",
			Policy:      policiesToUnset,
			UserOrGroup: userOrGroup,
			IsGroup:     isGroup,
		})
	} else {
		fatalIf(probe.NewError(e), "Unable to unset the policy")
	}
	return nil
}
