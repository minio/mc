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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPLdapUpdateCmd = cli.Command{
	Name:         "update",
	Usage:        "Update an LDAP IDP configuration",
	Action:       mainAdminIDPLDAPUpdate,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [CFG_PARAMS...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Update the LDAP IDP configuration.
     {{.Prompt}} {{.HelpName}} play/ \
          lookup_bind_dn=cn=admin,dc=min,dc=io \
          lookup_bind_password=somesecret
`,
}

func mainAdminIDPLDAPUpdate(ctx *cli.Context) error {
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

	if cfgName != madmin.Default {
		fatalIf(probe.NewError(errors.New("all config parameters must be of the form \"key=value\"")),
			"Bad LDAP IDP configuration")
	}

	inputCfg := strings.Join(input, " ")

	restart, e := client.AddOrUpdateIDPConfig(globalContext, madmin.LDAPIDPCfg, cfgName, inputCfg, true)
	fatalIf(probe.NewError(e), "Unable to update LDAP IDP configuration")

	// Print set config result
	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
