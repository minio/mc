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

var adminClusterSubcommands = []cli.Command{
	adminClusterBucketCmd,
	adminClusterIAMCmd,
}

var adminClusterCmd = cli.Command{
	Name:            "cluster",
	Usage:           "manage MinIO cluster metadata",
	Action:          mainAdminCluster,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     adminClusterSubcommands,
	HideHelpCommand: true,
}

// mainAdminCluster is the handle for "mc admin cluster" command.
func mainAdminCluster(ctx *cli.Context) error {
	commandNotFound(ctx, adminClusterSubcommands)
	return nil
	// Sub-commands like "bucket", "iam" have their own main.
}
