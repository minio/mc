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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminUserInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display info of a user",
	Action:       mainAdminUserInfo,
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
  1. Display the info of a user "foobar".
     {{.Prompt}} {{.HelpName}} myminio foobar
`,
}

// checkAdminUserAddSyntax - validate all the passed arguments
func checkAdminUserInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainAdminUserInfo is the handler for "mc admin user info" command.
func mainAdminUserInfo(ctx *cli.Context) error {
	checkAdminUserInfoSyntax(ctx)

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	user, e := client.GetUserInfo(globalContext, args.Get(1))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get user info")

	memberOf := []userGroup{}
	for _, group := range user.MemberOf {
		gd, e := client.GetGroupDescription(globalContext, group)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch group info")
		policies := []string{}
		if gd.Policy != "" {
			policies = strings.Split(gd.Policy, ",")
		}
		memberOf = append(memberOf, userGroup{
			Name:     gd.Name,
			Policies: policies,
		})
	}

	printMsg(userMessage{
		op:             ctx.Command.Name,
		AccessKey:      args.Get(1),
		PolicyName:     user.PolicyName,
		UserStatus:     string(user.Status),
		MemberOf:       memberOf,
		Authentication: authInfoToUserMessage(user.AuthInfo),
	})

	return nil
}

func authInfoToUserMessage(a *madmin.UserAuthInfo) string {
	if a == nil {
		return ""
	}

	authServer := ""
	if a.Type != madmin.BuiltinUserAuthType {
		authServer = "/" + a.AuthServer
	}

	return fmt.Sprintf("%s%s (%s)", a.Type, authServer, a.AuthServerUserID)
}
