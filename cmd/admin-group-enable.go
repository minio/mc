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

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminGroupEnableCmd = cli.Command{
	Name:   "enable",
	Usage:  "enable a group",
	Action: mainAdminGroupEnableDisable,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET GROUPNAME

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable group 'allcents'.
     {{.Prompt}} {{.HelpName}} myminio allcents
`,
}

// checkAdminGroupEnableSyntax - validate all the passed arguments
func checkAdminGroupEnableSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
}

// mainAdminGroupEnableDisable is the handle for "mc admin group enable|disable" command.
func mainAdminGroupEnableDisable(ctx *cli.Context) error {
	checkAdminGroupEnableSyntax(ctx)

	console.SetColor("GroupMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	group := args.Get(1)
	var err1 error
	var status madmin.GroupStatus
	if ctx.Command.Name == "enable" {
		status = madmin.GroupEnabled
	} else if ctx.Command.Name == "disable" {
		status = madmin.GroupDisabled
	} else {
		err1 = errors.New("cannot happen")
		fatalIf(probe.NewError(err1).Trace(args...), "Could not get group enable")
	}
	err1 = client.SetGroupStatus(globalContext, group, status)
	fatalIf(probe.NewError(err1).Trace(args...), "Could not get group enable")

	printMsg(groupMessage{
		op:          ctx.Command.Name,
		GroupName:   group,
		GroupStatus: string(status),
	})

	return nil
}
