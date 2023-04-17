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
	"github.com/minio/mc/pkg/probe"
)

var adminIDPOpenidRemoveCmd = cli.Command{
	Name:         "remove",
	ShortName:    "rm",
	Usage:        "remove OpenID IDP server configuration",
	Action:       mainAdminIDPOpenIDRemove,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [CFG_NAME]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove the default OpenID IDP configuration (CFG_NAME is omitted).
     {{.Prompt}} {{.HelpName}} play/
  2. Remove OpenID IDP configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ dex_test
`,
}

func mainAdminIDPOpenIDRemove(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1)
	}

	args := ctx.Args()

	var cfgName string
	if len(args) == 2 {
		cfgName = args.Get(1)
	}
	return adminIDPRemove(ctx, true, cfgName)
}

func adminIDPRemove(ctx *cli.Context, isOpenID bool, cfgName string) error {
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	idpType := madmin.LDAPIDPCfg
	if isOpenID {
		idpType = madmin.OpenidIDPCfg
	}

	restart, e := client.DeleteIDPConfig(globalContext, idpType, cfgName)
	fatalIf(probe.NewError(e), "Unable to remove %s IDP config '%s'", idpType, cfgName)

	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
