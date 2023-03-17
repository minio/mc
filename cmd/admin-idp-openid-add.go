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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPOpenidAddCmd = cli.Command{
	Name:         "add",
	Usage:        "Create an OpenID IDP server configuration",
	Action:       mainAdminIDPOpenIDAdd,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [CFG_NAME] [CFG_PARAMS...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a default OpenID IDP configuration (CFG_NAME is omitted).
     {{.Prompt}} {{.HelpName}} play/ \
          client_id=minio-client-app \
          client_secret=minio-client-app-secret \
          config_url="http://localhost:5556/dex/.well-known/openid-configuration" \
          scopes="openid,groups" \
          redirect_uri="http://127.0.0.1:10000/oauth_callback" \
          role_policy="consoleAdmin"
  2. Create OpenID IDP configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ dex_test \
          client_id=minio-client-app \
          client_secret=minio-client-app-secret \
          config_url="http://localhost:5556/dex/.well-known/openid-configuration" \
          scopes="openid,groups" \
          redirect_uri="http://127.0.0.1:10000/oauth_callback" \
          role_policy="consoleAdmin"
`,
}

func mainAdminIDPOpenIDAdd(ctx *cli.Context) error {
	return mainAdminIDPOpenIDAddOrUpdate(ctx, false)
}

func mainAdminIDPOpenIDAddOrUpdate(ctx *cli.Context, update bool) error {
	if len(ctx.Args()) < 2 {
		showCommandHelpAndExit(ctx, 1)
	}

	args := ctx.Args()

	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	cfgName := madmin.Default
	input := args[1:]
	if !strings.Contains(args.Get(1), "=") {
		cfgName = args.Get(1)
		input = args[2:]
	}

	inputCfg := strings.Join(input, " ")

	restart, e := client.AddOrUpdateIDPConfig(globalContext, madmin.OpenidIDPCfg, cfgName, inputCfg, update)
	fatalIf(probe.NewError(e), "Unable to add OpenID IDP config to server")

	// Print set config result
	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
