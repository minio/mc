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
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPOpenidEnableCmd = cli.Command{
	Name:         "enable",
	Usage:        "enable an OpenID IDP server configuration",
	Action:       mainAdminIDPOpenIDEnable,
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
  1. Enable the default OpenID IDP configuration (CFG_NAME is omitted).
     {{.Prompt}} {{.HelpName}} play/
  2. Enable OpenID IDP configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ dex_test
`,
}

func mainAdminIDPOpenIDEnable(ctx *cli.Context) error {
	isOpenID, enable := true, true
	return adminIDPEnableDisable(ctx, isOpenID, enable)
}

func adminIDPEnableDisable(ctx *cli.Context, isOpenID bool, enable bool) error {
	if len(ctx.Args()) < 1 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1)
	}

	args := ctx.Args()
	cfgName := madmin.Default
	if len(args) == 2 {
		cfgName = args.Get(2)
	}
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	idpType := madmin.LDAPIDPCfg
	if isOpenID {
		idpType = madmin.OpenidIDPCfg
	}

	configBody := "enable=on"
	if !enable {
		configBody = "enable=off"
	}

	restart, e := client.AddOrUpdateIDPConfig(globalContext, idpType, cfgName, configBody, true)
	fatalIf(probe.NewError(e), "Unable to remove %s IDP config '%s'", idpType, cfgName)

	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
