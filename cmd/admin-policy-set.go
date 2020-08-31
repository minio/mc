/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminPolicySetCmd = cli.Command{
	Name:   "set",
	Usage:  "set IAM policy on a user or group",
	Action: mainAdminPolicySet,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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
  1. Set the "readwrite" policy for user "james".
     {{.Prompt}} {{.HelpName}} myminio readwrite user=james

  2. Set the "readonly" policy for group "auditors".
     {{.Prompt}} {{.HelpName}} myminio readonly group=auditors
`,
}

var (
	errBadUserGroupArg = errors.New("Last argument must be of the form user=xx or group=xx")
)

func checkAdminPolicySetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
}

func parseEntityArg(arg string) (userOrGroup string, isGroup bool, err error) {
	parts := strings.Split(arg, "=")
	switch {
	case len(parts) != 2:
		fallthrough
	case parts[1] == "":
		err = errBadUserGroupArg
	case strings.ToLower(parts[0]) == "user":
		userOrGroup = parts[1]
		isGroup = false
	case strings.ToLower(parts[0]) == "group":
		userOrGroup = parts[1]
		isGroup = true
	default:
		err = errBadUserGroupArg

	}
	return
}

// mainAdminPolicySet is the handler for "mc admin policy set" command.
func mainAdminPolicySet(ctx *cli.Context) error {
	checkAdminPolicySetSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))
	console.SetColor("Policy", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	policyName := args.Get(1)
	entityArg := args.Get(2)

	userOrGroup, isGroup, e1 := parseEntityArg(entityArg)
	fatalIf(probe.NewError(e1).Trace(args...), "Bad last argument")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.SetPolicy(globalContext, policyName, userOrGroup, isGroup)

	if e == nil {
		printMsg(userPolicyMessage{
			op:          "set",
			Policy:      policyName,
			UserOrGroup: userOrGroup,
			IsGroup:     isGroup,
		})
	} else {
		fatalIf(probe.NewError(e), "Unable to set the policy")
	}
	return nil
}
