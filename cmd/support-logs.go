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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var logsFlags = append(globalFlags, cli.BoolFlag{
	Name:   "dev",
	Usage:  "development mode - talks to local SUBNET",
	Hidden: true,
})

var supportLogsCmd = cli.Command{
	Name:         "logs",
	Usage:        "configure logs settings",
	OnUsageError: onUsageError,
	Action:       mainLogs,
	Before:       setGlobalsFromContext,
	Flags:        logsFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS enable|disable|status

OPTIONS:
  enable - Enable pushing MinIO logs to SUBNET in real-time
  disable - Disable pushing MinIO logs to SUBNET
  status - Display logs settings

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable logs for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play enable

  2. Disable logs for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play disable

  3. Check logs status for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play status
`,
}

func mainLogs(ctx *cli.Context) error {
	checkToggleCmdSyntax(ctx, "logs")

	aliasedURL := ctx.Args().Get(0)
	arg := ctx.Args().Get(1)
	fatalIf(probe.NewError(validateToggleCmdArg(arg)), "Invalid arguments.")

	if arg == "status" {
		printToggleFeatureStatus(aliasedURL, "logger_webhook", "logger_webhook:subnet")
		return nil
	}

	enable := arg == "enable"
	configureSubnetWebhook(aliasedURL, enable)

	return nil
}

func configureSubnetWebhook(alias string, enable bool) {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(alias)
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
		targetAlias: alias,
		restart:     restart,
	})
}
