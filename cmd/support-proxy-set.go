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
	"net/url"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

type supportProxySetMessage struct {
	Status string `json:"status"`
	Proxy  string `json:"proxy"`
}

// String colorized proxy set message
func (s supportProxySetMessage) String() string {
	return console.Colorize(supportSuccessMsgTag, "Proxy is now set to "+s.Proxy)
}

// JSON jsonified proxy set message
func (s supportProxySetMessage) JSON() string {
	s.Status = "success"
	return toJSON(s)
}

var supportProxySetCmd = cli.Command{
	Name:            "set",
	Usage:           "configure proxy to given URL",
	Action:          mainSupportProxySet,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET PROXY_URL

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set the proxy to http://my.proxy for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} myminio http://my.proxy
`,
}

func checkSupportProxySetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainSupportProxySet is the handle for "mc support proxy set" command.
func mainSupportProxySet(ctx *cli.Context) error {
	// Check for command syntax
	checkSupportProxySetSyntax(ctx)
	setSuccessMessageColor()

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)

	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Call set config API
	proxy := args.Get(1)
	if len(proxy) == 0 {
		fatal(errDummy().Trace(), "Proxy must not be empty")
	}

	_, e := url.Parse(proxy)
	fatalIf(probe.NewError(e), "Invalid proxy:")

	// Main execution
	_, e = client.SetConfigKV(globalContext, "subnet proxy="+proxy)
	fatalIf(probe.NewError(e), "Unable to set proxy '%s':", proxy)

	printMsg(supportProxySetMessage{Proxy: proxy})
	return nil
}
