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

var adminIDPInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "Show IDP server config info",
	Before:       setGlobalsFromContext,
	Action:       mainAdminIDPGet,
	OnUsageError: onUsageError,
	Hidden:       true,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ID_TYPE [CFG_NAME]

  ID_TYPE must be one of 'ldap' or 'openid'.

  **DEPRECATED**: This command will be removed in a future version. Please use
  "mc admin idp ldap|openid" instead.

FLAGS:
   {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show configuration info for default (un-named) openid configuration.
     {{.Prompt}} {{.HelpName}} play/ openid
  2. Show configuration info for openid configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ openid dex_test
  3. Show configuration info for ldap.
     {{.Prompt}} {{.HelpName}} play/ ldap
`,
}

func mainAdminIDPGet(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 || len(ctx.Args()) > 3 {
		showCommandHelpAndExit(ctx, 1)
	}

	args := ctx.Args()
	idpType := args.Get(1)
	validateIDType(idpType)
	isOpenID := idpType == madmin.OpenidIDPCfg

	var cfgName string
	if len(args) == 3 {
		cfgName = args.Get(2)
	}

	return adminIDPInfo(ctx, isOpenID, cfgName)
}
