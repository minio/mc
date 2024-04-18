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

var adminQuotaFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "hard",
		Usage: "set a hard quota, disallowing writes after quota is reached",
	},
	cli.BoolFlag{
		Name:  "clear",
		Usage: "clears bucket quota configured for bucket",
	},
}

var adminBucketQuotaCmd = cli.Command{
	Name:            "quota",
	Usage:           "manage bucket quota",
	Action:          mainAdminBucketQuota,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminQuotaFlags, globalFlags...),
	HideHelpCommand: true,
}

// mainAdminBucketQuota is the handler for "mc admin bucket quota" command.
func mainAdminBucketQuota(_ *cli.Context) error {
	deprecatedError("mc quota")
	return nil
}
