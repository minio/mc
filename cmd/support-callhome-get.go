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

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var callhomeGetCmd = cli.Command{
	Name:         "get",
	Usage:        "retrieve callhome settings",
	OnUsageError: onUsageError,
	Action:       mainCallhomeGet,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET option

OPTIONS:
  logs - push MinIO server logs to SUBNET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get all callhome settings for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play

  2. Get callhome 'logs' setting for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play logs
`,
}

// checkCallhomeSyntax - validate arguments passed by a user
func checkCallhomeGetSyntax(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 || nArgs > 2 {
		cli.ShowCommandHelpAndExit(ctx, "get", 1) // last argument is exit code
	}
}

func validateCallhomeGetArgs(opt string) error {
	so := supportedOptions()
	if !so.Contains(opt) {
		return fmt.Errorf("Invalid option %s. Valid options are %s", opt, so.String())
	}
	return nil
}

// callhomeGetMessage - container to hold callhome setting information
type callhomeGetMessage struct {
	Opt   string `json:"option"`
	Value string `json:"value"`
}

// String colorized service status message.
func (m callhomeGetMessage) String() string {
	return m.Opt + "=" + string(m.Value)
}

// JSON jsonified service status Message message.
func (m callhomeGetMessage) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

func printCallhomeSetting(alias string, s string) {
	switch s {
	case "logs":
		// Create a new MinIO Admin Client
		client, err := newAdminClient(alias)
		fatalIf(err, "Unable to initialize admin connection.")

		val := "off"

		kvs := getSubSysKeyFromMinIOConfig(client, "logger_webhook:subnet")
		ep, found := kvs.Lookup("endpoint")

		if found && len(ep) > 0 {
			// subnet webhook is configured. check if it is enabled
			enable, found := kvs.Lookup("enable")

			if found {
				val = enable
			} else {
				// if the 'enable' key is not found, it means that the webhook is enabled
				val = "on"
			}
		}

		printMsg(callhomeGetMessage{
			Opt:   s,
			Value: val,
		})
	}
}

func mainCallhomeGet(ctx *cli.Context) error {
	checkCallhomeGetSyntax(ctx)

	args := ctx.Args()
	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)
	var opt string
	if len(args) == 2 {
		opt = args.Get(1)
		fatalIf(probe.NewError(validateCallhomeGetArgs(opt)), fmt.Sprintf("Invalid arguments: %s", strings.Join(args, ",")))
	}

	if opt == "" {
		// print all settings
		for _, o := range supportedOptions().ToSlice() {
			printCallhomeSetting(alias, o)
		}
	} else {
		printCallhomeSetting(alias, opt)
	}

	return nil
}
