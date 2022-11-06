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
  {{.HelpName}} TARGET ID_TYPE [CFG_NAME] [CFG_PARAMS...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create/Update the default LDAP IDP configuration (CFG_NAME is omitted).
     {{.Prompt}} {{.HelpName}} play/ ldap \
          lookup_bind_dn=cn=admin,dc=min,dc=io \
          lookup_bind_password=somesecret
`,
}

func mainAdminIDPLDAPUpdate(ctx *cli.Context) error {
	return nil
}
