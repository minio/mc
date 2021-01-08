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

import "github.com/minio/cli"

var (
	adminFlags = []cli.Flag{}
)

const (
	// dot represents a list item, for eg. server status - online (green) or offline (red)
	dot = "●"
	// check represents successful operation
	check = "✔"
)

var adminCmdSubcommands = []cli.Command{
	adminServiceCmd,
	adminServerUpdateCmd,
	adminInfoCmd,
	adminUserCmd,
	adminGroupCmd,
	adminPolicyCmd,
	adminConfigCmd,
	adminHealCmd,
	adminProfileCmd,
	adminTopCmd,
	adminTraceCmd,
	adminConsoleCmd,
	adminPrometheusCmd,
	adminKMSCmd,
	adminHealthCmd,
	adminSubnetCmd,
	adminBucketCmd,
	adminTierCmd,
}

var adminCmd = cli.Command{
	Name:            "admin",
	Usage:           "manage MinIO servers",
	Action:          mainAdmin,
	Subcommands:     adminCmdSubcommands,
	HideHelpCommand: true,
	Before:          setGlobalsFromContext,
	Flags:           append(adminFlags, globalFlags...),
}

const dateTimeFormatFilename = "2006-01-02T15-04-05.999999-07-00"

// mainAdmin is the handle for "mc admin" command.
func mainAdmin(ctx *cli.Context) error {
	commandNotFound(ctx, adminCmdSubcommands)
	return nil
	// Sub-commands like "service", "heal", "top" have their own main.
}
