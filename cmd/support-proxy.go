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

var supportProxySubcommands = []cli.Command{
	supportProxySetCmd,
	supportProxyRemoveCmd,
	supportProxyShowCmd,
}

var supportProxyCmd = cli.Command{
	Name:            "proxy",
	Usage:           "configure proxy",
	Action:          mainSupportProxy,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           supportGlobalFlags,
	Subcommands:     supportProxySubcommands,
	HideHelpCommand: true,
}

// mainSupportProxy is the handler for "mc support proxy" command.
func mainSupportProxy(ctx *cli.Context) error {
	commandNotFound(ctx, supportProxySubcommands)
	return nil
	// Sub-commands like "set", "remove", "show" have their own main.
	// Check for command syntax
}
