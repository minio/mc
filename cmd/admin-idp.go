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

var adminIDPCmd = cli.Command{
	Name:            "idp",
	Usage:           "manage MinIO IDentity Provider server configuration",
	Action:          mainAdminIDP,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	Hidden:          true,
	CustomHelpTemplate: `This command's functionality has moved and this command is DEPRECATED.

Please use commands under 'mc idp ldap|openid' instead.
`,
}

func mainAdminIDP(_ *cli.Context) error {
	deprecatedError("mc idp ldap|openid")
	return nil
}
