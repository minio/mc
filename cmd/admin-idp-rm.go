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
	"errors"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPRmCmd = cli.Command{
	Name:         "rm",
	Usage:        "Remove an IDP configuration",
	Before:       setGlobalsFromContext,
	Action:       mainAdminIDPRemove,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET IDP_TYPE CFG_NAME

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove an OpenID configuration from the server.
     {{.Prompt}} {{.HelpName}} play/ openid myidp
`,
}

func mainAdminIDPRemove(ctx *cli.Context) error {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, "rm", 1)
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	idpType := args.Get(1)

	if idpType != "openid" {
		fatalIf(probe.NewError(errors.New("not implemented")), "This feature is not yet available")
	}

	cfgName := args.Get(2)

	restart, e := client.DeleteIDPConfig(globalContext, idpType, cfgName)
	fatalIf(probe.NewError(e), "Unable to remove %s IDP config '%s'", idpType, cfgName)

	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
