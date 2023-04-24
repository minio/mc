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

var adminTopAPIFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "name",
		Usage: "summarize current calls for matching API name",
	},
	cli.StringSliceFlag{
		Name:  "path",
		Usage: "summarize current API calls only on matching path",
	},
	cli.StringSliceFlag{
		Name:  "node",
		Usage: "summarize current API calls only on matching servers",
	},
	cli.BoolFlag{
		Name:  "errors, e",
		Usage: "summarize current API calls throwing only errors",
	},
}

var adminTopAPICmd = cli.Command{
	Name:            "api",
	Usage:           "summarize API events on MinIO server in real-time",
	Action:          mainAdminTopAPI,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminTopAPIFlags, globalFlags...),
	Hidden:          true,
	HideHelpCommand: true,
	CustomHelpTemplate: `Please use 'mc support top api' 
`,
}

func mainAdminTopAPI(_ *cli.Context) error {
	deprecatedError("mc support top api")
	return nil
}
