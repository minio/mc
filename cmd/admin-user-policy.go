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
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	iampolicy "github.com/minio/minio/pkg/iam/policy"
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
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code
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

	var combinedPolicy iampolicy.Policy

	policies := strings.Split(user.PolicyName, ",")

	for _, p := range policies {
		buf, e := client.InfoCannedPolicy(globalContext, p)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch user policy document")
		policy, e := iampolicy.ParseConfig(bytes.NewReader(buf))
		fatalIf(probe.NewError(e).Trace(args...), "Unable to parse user policy document")
		combinedPolicy = combinedPolicy.Merge(*policy)
	}

	var jsoniter = jsoniter.ConfigCompatibleWithStandardLibrary
	policyJSON, e := jsoniter.MarshalIndent(combinedPolicy, "", "   ")
	fatalIf(probe.NewError(e).Trace(args...), "Unable to parse user policy document")

	fmt.Println(string(policyJSON))

	return nil
}
