// Copyright (c) 2015-2021 MinIO, Inc.
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

var adminConsoleCmd = cli.Command{
	Name:            "console",
	Usage:           "show console logs for MinIO server",
	Action:          mainAdminConsole,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	Hidden:          true, // deprecated June 2022
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [NODENAME]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show console logs for a MinIO server with alias 'play'
     {{.Prompt}} {{.HelpName}} play

  2. Show last 5 log entries for node 'node1' on MinIO server with alias 'myminio'
     {{.Prompt}} {{.HelpName}} --limit 5 myminio node1

  3. Show application error logs on MinIO server with alias 'play'
     {{.Prompt}} {{.HelpName}} --type application play
`,
}

// mainAdminConsole - the entry function of console command
func mainAdminConsole(ctx *cli.Context) error {
	return nil
}
