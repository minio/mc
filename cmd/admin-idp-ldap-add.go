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

var adminIDPLdapAddCmd = cli.Command{
	Name:         "add",
	Usage:        "Create an LDAP IDP server configuration",
	Action:       mainAdminIDPLDAPAdd,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [CFG_PARAMS...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create LDAP IDentity Provider configuration.
     {{.Prompt}} {{.HelpName}} myminio/ \
          server_addr=myldapserver:636 \
          lookup_bind_dn=cn=admin,dc=min,dc=io \
          lookup_bind_password=somesecret \
          user_dn_search_base_dn=dc=min,dc=io \
          user_dn_search_filter="(uid=%s)" \
          group_search_base_dn=ou=swengg,dc=min,dc=io \
          group_search_filter="(&(objectclass=groupofnames)(member=%d))"
`,
}

func mainAdminIDPLDAPAdd(ctx *cli.Context) error {
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

	restart, e := client.AddOrUpdateIDPConfig(globalContext, madmin.LDAPIDPCfg, cfgName, inputCfg, false)
	fatalIf(probe.NewError(e), "Unable to add LDAP IDP config to server")

	// Print set config result
	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
