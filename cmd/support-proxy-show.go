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
	"github.com/minio/pkg/v3/console"
)

var supportProxyShowCmd = cli.Command{
	Name:            "show",
	Usage:           "Show the configured proxy",
	Action:          mainSupportProxyShow,
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
  1. Show the proxy configured for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} myminio
`,
}

type supportProxyShowMessage struct {
	Status string `json:"status"`
	Proxy  string `json:"proxy"`
}

// String colorized proxy show message
func (s supportProxyShowMessage) String() string {
	msg := s.Proxy
	if len(msg) == 0 {
		msg = "Proxy is not configured"
	}
	return console.Colorize(supportSuccessMsgTag, msg)
}

// JSON jsonified proxy show message
func (s supportProxyShowMessage) JSON() string {
	s.Status = "success"
	return toJSON(s)
}

func checkSupportProxyShowSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainSupportProxyShow is the handler for "mc support proxy show" command.
func mainSupportProxyShow(ctx *cli.Context) error {
	// Check for command syntax
	checkSupportProxyShowSyntax(ctx)
	setSuccessMessageColor()

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)

	validateClusterRegistered(alias, false)

	// Main execution
	// get the subnet proxy config from MinIO if available
	proxy, supported := getKeyFromSubnetConfig(alias, "proxy")
	if !supported {
		fatal(errDummy().Trace(), "Proxy configuration not supported in this version of MinIO.")
	}

	printMsg(supportProxyShowMessage{Proxy: proxy})
	return nil
}
