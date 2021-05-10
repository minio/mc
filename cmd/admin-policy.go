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

var adminPolicySubcommands = []cli.Command{
	adminPolicyAddCmd,
	adminPolicyRemoveCmd,
	adminPolicyListCmd,
	adminPolicyInfoCmd,
	adminPolicySetCmd,
	adminPolicyUnsetCmd,
	adminPolicyUpdateCmd,
}

var adminPolicyCmd = cli.Command{
	Name:            "policy",
	Usage:           "manage policies defined in the MinIO server",
	Action:          mainAdminPolicy,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     adminPolicySubcommands,
	HideHelpCommand: true,
}

// mainAdminPolicy is the handle for "mc admin policy" command.
func mainAdminPolicy(ctx *cli.Context) error {
	commandNotFound(ctx, adminPolicySubcommands)
	return nil
	// Sub-commands like "get", "set" have their own main.
}
