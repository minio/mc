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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
)

var callhomeFlags = append(globalFlags, cli.BoolFlag{
	Name:   "dev",
	Usage:  "development mode - talks to local SUBNET",
	Hidden: true,
})

var callhomeSetCmd = cli.Command{
	Name:         "set",
	Usage:        "configure callhome settings",
	OnUsageError: onUsageError,
	Action:       mainCallhomeSet,
	Before:       setGlobalsFromContext,
	Flags:        callhomeFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET option=on|off

OPTIONS:
  logs - push MinIO server logs to SUBNET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable logs callhome for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play logs=on

  2. Disable logs callhome for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play logs=off
`,
}

func supportedOptions() set.StringSet {
	return set.CreateStringSet("logs")
}

func validOptionValues() set.StringSet {
	return set.CreateStringSet("on", "off")
}

// checkCallhomeSyntax - validate arguments passed by a user
func checkCallhomeSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
}

func validateCallhomeSetArgs(args []string) error {
	for _, arg := range args {
		_, _, e := getOptionKV(arg)
		if e != nil {
			return e
		}
	}
	return nil
}

func mainCallhomeSet(ctx *cli.Context) error {
	checkCallhomeSetSyntax(ctx)

	aliasedURL := ctx.Args().Get(0)
	args := ctx.Args().Tail()
	fatalIf(probe.NewError(validateCallhomeSetArgs(args)), fmt.Sprintf("Invalid arguments: %s", strings.Join(args, ",")))

	for _, option := range ctx.Args().Tail() {
		// options have already been validated, so third (error) return value can be ignored
		k, v, _ := getOptionKV(option)
		if k == "logs" {
			configureSubnetWebhook(aliasedURL, v == "on")
		}
	}

	return nil
}

func configureSubnetWebhook(aliasedURL string, enable bool) {
	// Get the alias parameter from cli
	alias, _ := url2Alias(aliasedURL)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	apiKey := getSubnetAPIKeyFromConfig(alias)
	if len(apiKey) == 0 {
		e := fmt.Errorf("Please register the cluster first by running 'mc support register %s'", alias)
		fatalIf(probe.NewError(e), "Cluster not registered.")
	}

	enableStr := "off"
	if enable {
		enableStr = "on"
	}

	input := fmt.Sprintf("logger_webhook:subnet endpoint=%s auth_token=%s enable=%s",
		subnetLogWebhookURL(), apiKey, enableStr)

	// Call set config API
	restart, e := client.SetConfigKV(globalContext, input)
	fatalIf(probe.NewError(e), "Unable to set '%s' to server", input)

	// Print set config result
	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})
}

func getOptionKV(input string) (string, string, error) {
	so := supportedOptions()
	vv := validOptionValues()
	parts := strings.Split(input, "=")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid argument %s. Must be of the form '%s=%s'",
			input, strings.Join(so.ToSlice(), "|"), strings.Join(vv.ToSlice(), "|"))
	}

	opt := parts[0]
	if !so.Contains(opt) {
		return "", "", fmt.Errorf("Invalid option %s. Valid options are %s", opt, so.String())
	}

	val := parts[1]
	if !vv.Contains(val) {
		return "", "", fmt.Errorf("Invalid value %s for option %s. Valid values are %s", val, opt, vv.String())
	}

	return opt, val, nil
}
