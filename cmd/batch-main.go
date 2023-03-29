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

var batchSubcommands = []cli.Command{
	batchGenerateCmd,
	batchStartCmd,
	batchListCmd,
	batchStatusCmd,
	batchDescribeCmd,
	// batchSuspendResumeCmd,
	batchCancelCmd,
}

var batchCmd = cli.Command{
	Name:            "batch",
	Usage:           "manage batch jobs",
	Action:          mainBatch,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Subcommands:     batchSubcommands,
	HideHelpCommand: true,
}

// mainBatch is the handle for "mc batch" command.
func mainBatch(ctx *cli.Context) error {
	commandNotFound(ctx, batchSubcommands)
	return nil
	// Sub-commands like "generate", "list", "info" have their own main.
}
