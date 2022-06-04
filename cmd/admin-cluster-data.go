// Copyright (c) 2022 MinIO, Inc.
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

var adminClusterDataSubcommands = []cli.Command{
	adminClusterDataListCmd,
	adminClusterDataCopyCmd,
}

var adminClusterDataCmd = cli.Command{
	Name:            "data",
	Usage:           "manage data migration on MinIO cluster",
	Action:          mainAdminClusterData,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     adminClusterDataSubcommands,
	HideHelpCommand: true,
}

// mainAdminClusterData is the handle for "mc admin cluster data" command.
func mainAdminClusterData(ctx *cli.Context) error {
	commandNotFound(ctx, adminClusterDataSubcommands)
	return nil
	// Sub-commands like "ls", "copy" have their own main.
}
