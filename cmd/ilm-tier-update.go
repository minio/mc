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

var ilmTierUpdateCmd = cli.Command{
	Name:         "update",
	Usage:        "update an existing remote tier configuration",
	Action:       mainAdminTierEdit,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminTierEditFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS NAME

NAME:
  Name of remote tier. e.g WARM-TIER

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Update credentials for an existing Azure Blob Storage remote tier:
     {{.Prompt}} {{.HelpName}} myminio AZTIER --account-key ACCOUNT-KEY

  2. Update credentials for an existing AWS S3 compatible remote tier:
     {{.Prompt}} {{.HelpName}} myminio S3TIER --access-key ACCESS-KEY --secret-key SECRET-KEY

  3. Update credentials for an existing Google Cloud Storage remote tier:
     {{.Prompt}} {{.HelpName}} myminio GCSTIER --credentials-file /path/to/credentials.json
`,
}
