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
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var supportLogsSubcommands = []cli.Command{
	supportLogsEnableCmd,
	supportLogsDisableCmd,
	supportLogsStatusCmd,
	supportLogsShowCmd,
}

var logsConfigureFlags = append(globalFlags, cli.BoolFlag{
	Name:   "dev",
	Usage:  "development mode - talks to local SUBNET",
	Hidden: true,
})

var supportLogsCmd = cli.Command{
	Name:            "logs",
	Usage:           "configure/display MinIO console logs",
	OnUsageError:    onUsageError,
	Action:          mainLogs,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	Subcommands:     supportLogsSubcommands,
}

type supportLogsMessage struct {
	Status string `json:"status"`
	Logs   string `json:"logs"`
	MsgPfx string `json:"-"`
}

// String colorized service status message.
func (s supportLogsMessage) String() string {
	return console.Colorize(featureToggleMessageTag, s.MsgPfx+s.Logs)
}

// JSON jsonified service status message.
func (s supportLogsMessage) JSON() string {
	s.Status = "success"
	jsonBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func configureSubnetWebhook(alias string, enable bool) {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	apiKey := validateClusterRegistered(alias)

	enableStr := "off"
	if enable {
		enableStr = "on"
	}

	input := fmt.Sprintf("logger_webhook:subnet endpoint=%s auth_token=%s enable=%s",
		subnetLogWebhookURL(), apiKey, enableStr)

	// Call set config API
	_, e := client.SetConfigKV(globalContext, input)
	fatalIf(probe.NewError(e), "Unable to set '%s' to server", input)

	printMsg(supportLogsMessage{
		Logs:   featureStatusStr(enable),
		MsgPfx: "Logging to support is now ",
	})
}

func validateLogsToggleCmd(ctx *cli.Context, cmdName string) string {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, cmdName, 1) // last argument is exit code
	}
	alias, _ := url2Alias(ctx.Args().Get(0))
	return alias
}

// mainLogs is the handle for "mc support logs" command.
func mainLogs(ctx *cli.Context) error {
	commandNotFound(ctx, supportLogsSubcommands)
	return nil
	// Sub-commands like "enable", "disable", "status", "print" have their own main.
}
