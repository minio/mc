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
	"github.com/minio/madmin-go/v2"
)

var adminIDPRmCmd = cli.Command{
	Name:         "rm",
	Usage:        "Remove an IDP configuration",
	Before:       setGlobalsFromContext,
	Action:       mainAdminIDPRemove,
	Hidden:       true,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET IDP_TYPE CFG_NAME

  **DEPRECATED**: This command will be removed in a future version. Please use
  "mc admin idp ldap|openid" instead.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove an OpenID configuration from the server.
     {{.Prompt}} {{.HelpName}} play/ openid myidp
  2. Remove default LDAP configuration from the server.
     {{.Prompt}} {{.HelpName}} play/ ldap _
`,
}

func mainAdminIDPRemove(ctx *cli.Context) error {
	if len(ctx.Args()) != 3 {
		showCommandHelpAndExit(ctx, 1)
	}

	args := ctx.Args()
	idpType := args.Get(1)
	validateIDType(idpType)

	cfgName := args.Get(2)
	return adminIDPRemove(ctx, idpType == madmin.OpenidIDPCfg, cfgName)
}
