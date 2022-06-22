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
)

var supportLogsStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "check logs status for cluster with alias 'play'",
	OnUsageError: onUsageError,
	Action:       mainStatusLogs,
	Before:       setGlobalsFromContext,
	Flags:        logsConfigureFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
USAGE:
  {{.HelpName}} ALIAS
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
	1. Check logs status for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play
`,
}

func mainStatusLogs(ctx *cli.Context) error {
	setToggleMessageColor()
	alias := validateLogsToggleCmd(ctx, "status")
	enabled := isFeatureEnabled(alias, "logger_webhook", "logger_webhook:subnet")
	printMsg(supportLogsMessage{
		Logs: featureStatusStr(enabled),
	})

	return nil
}
