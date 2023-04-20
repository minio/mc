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

var subnetHealthSubcommands = []cli.Command{
	adminSubnetHealthCmd,
	adminSubnetRegisterCmd,
}

var adminSubnetCmd = cli.Command{
	Name:        "subnet",
	Usage:       "Subnet related commands",
	Action:      mainAdminSubnet,
	Before:      setGlobalsFromContext,
	Flags:       globalFlags,
	Subcommands: subnetHealthSubcommands,
	Hidden:      true,
}

// mainAdminSubnet is the handle for "mc admin subnet" command.
func mainAdminSubnet(_ *cli.Context) error {
	deprecatedError("mc support")
	return nil
	// Sub-commands like "health", "register" have their own main.
}

func adminHealthCmd() cli.Command {
	cmd := adminSubnetHealthCmd
	cmd.Hidden = true
	return cmd
}
