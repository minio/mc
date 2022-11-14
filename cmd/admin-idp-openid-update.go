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

var adminIDPOpenidUpdateCmd = cli.Command{
	Name:         "update",
	Usage:        "Update an OpenID IDP configuration",
	Action:       mainAdminIDPOpenIDUpdate,
	Before:       setGlobalsFromContext,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [CFG_NAME] [CFG_PARAMS...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Update the default OpenID IDP configuration (CFG_NAME is omitted).
     {{.Prompt}} {{.HelpName}} play/
          scopes="openid,groups" \
          role_policy="consoleAdmin"
  2. Update configuration for OpenID IDP configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ dex_test \
          scopes="openid,groups" \
          role_policy="consoleAdmin"
`,
}

func mainAdminIDPOpenIDUpdate(ctx *cli.Context) error {
	return mainAdminIDPOpenIDAddOrUpdate(ctx, true)
}
