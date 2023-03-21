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

var topLocksFlag = []cli.Flag{
	cli.BoolFlag{
		Name:  "stale",
		Usage: "list stale locks",
	},
	cli.IntFlag{
		Name:   "count",
		Usage:  "number of top locks",
		Hidden: true,
		Value:  10,
	},
}

var adminTopLocksCmd = cli.Command{
	Name:         "locks",
	Usage:        "get a list of the 10 oldest locks on a MinIO cluster.",
	Before:       setGlobalsFromContext,
	Action:       mainAdminTopLocks,
	OnUsageError: onUsageError,
	Flags:        append(globalFlags, topLocksFlag...),
	CustomHelpTemplate: `Please use 'mc support top locks'
`,
}

func mainAdminTopLocks(_ *cli.Context) error {
	deprecatedError("mc support top locks")
	return nil
}
