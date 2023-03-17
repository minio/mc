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

import "github.com/minio/cli"

var replicateResyncSubcommands = []cli.Command{
	replicateResyncStartCmd,
	replicateResyncStatusCmd,
}

var replicateResyncCmd = cli.Command{
	Name:            "resync",
	Usage:           "re-replicate all previously replicated objects",
	HideHelpCommand: true,
	Action:          mainReplicateResync,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     replicateResyncSubcommands,
	Aliases:         []string{"reset"},
	HiddenAliases:   true,
}

// mainReplicateResync is the handle for "mc replicate resync" command.
func mainReplicateResync(ctx *cli.Context) error {
	commandNotFound(ctx, replicateResyncSubcommands)
	return nil
	// Sub-commands like "status", "start", have their own main.
}
