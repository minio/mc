// Copyright (c) 2015-2021 MinIO, Inc.
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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminPolicySetCmd = cli.Command{
	Name:         "set",
	Usage:        "set IAM policy on a user or group",
	Action:       mainAdminPolicySet,
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
	parts := strings.SplitN(arg, "=", 2)
	switch {
	case len(parts) != 2 || parts[1] == "":
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
	policyName := strings.TrimSpace(args.Get(1))
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
