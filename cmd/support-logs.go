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

var supportLogsCmd = cli.Command{
	Name:            "logs",
	Usage:           "show MinIO logs",
	OnUsageError:    onUsageError,
	Action:          mainLogsShowConsole,
	Before:          setGlobalsFromContext,
	Flags:           append(logsShowFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
USAGE:
  {{.HelpName}} [FLAGS] TARGET [NODENAME]
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show logs for a MinIO server with alias 'myminio'
     {{.Prompt}} {{.HelpName}} myminio
  2. Show last 5 log entries for node 'node1' for a MinIO server with alias 'myminio'
     {{.Prompt}} {{.HelpName}} --last 5 myminio node1
  3. Show application errors in logs for a MinIO server with alias 'myminio'
     {{.Prompt}} {{.HelpName}} --type application myminio
`,
}

func configureSubnetWebhook(alias string, enable bool) {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	var input string
	if enable {
		apiKey := validateClusterRegistered(alias, true)
		input = fmt.Sprintf("logger_webhook:subnet endpoint=%s auth_token=%s enable=on",
			subnetLogWebhookURL(), apiKey)
	} else {
		input = "logger_webhook:subnet enable=off"
	}

	// Call set config API
	_, e := client.SetConfigKV(globalContext, input)
	fatalIf(probe.NewError(e), "Unable to set '%s' to server", input)
}

func isLogsCallhomeEnabled(alias string) bool {
	return isFeatureEnabled(alias, "logger_webhook", "subnet")
}
