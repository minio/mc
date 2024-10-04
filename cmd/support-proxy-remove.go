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
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

type supportProxyRemoveMessage struct {
	Status string `json:"status"`
}

// String colorized proxy remove message
func (s supportProxyRemoveMessage) String() string {
	return console.Colorize(supportSuccessMsgTag, "Proxy has been removed")
}

// JSON jsonified proxy remove message
func (s supportProxyRemoveMessage) JSON() string {
	s.Status = "success"
	return toJSON(s)
}

var supportProxyRemoveCmd = cli.Command{
	Name:            "remove",
	Usage:           "Remove proxy configuration",
	Action:          mainSupportProxyRemove,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove the proxy configured for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} myminio
`,
}

func checkSupportProxyRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainSupportProxyRemove is the handler for "mc support proxy remove" command.
func mainSupportProxyRemove(ctx *cli.Context) error {
	// Check for command syntax
	checkSupportProxyRemoveSyntax(ctx)
	setSuccessMessageColor()

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)

	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Main execution
	_, e := client.DelConfigKV(globalContext, "subnet proxy")
	fatalIf(probe.NewError(e), "Unable to remove proxy:")

	printMsg(supportProxyRemoveMessage{})
	return nil
}
