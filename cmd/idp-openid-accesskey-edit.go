// Copyright (c) 2015-2024 MinIO, Inc.
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
)

var idpOpenIDAccesskeyEditFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "secret-key",
		Usage: "set a secret key for the  account",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "path to a JSON policy file",
	},
	cli.StringFlag{
		Name:  "name",
		Usage: "friendly name for the account",
	},
	cli.StringFlag{
		Name:  "description",
		Usage: "description for the account",
	},
	cli.StringFlag{
		Name:  "expiry-duration",
		Usage: "duration before the access key expires",
	},
	cli.StringFlag{
		Name:  "expiry",
		Usage: "expiry date for the access key",
	},
}

var idpOpenidAccesskeyEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "edit existing access keys for OpenID",
	Action:       mainIDPOpenIDAccesskeyEdit,
	Before:       setGlobalsFromContext,
	Flags:        append(idpOpenIDAccesskeyEditFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Change the secret key for the access key "testkey"
     {{.Prompt}} {{.HelpName}} myminio/ testkey --secret-key 'xxxxxxx'
  2. Change the expiry duration for the access key "testkey"
     {{.Prompt}} {{.HelpName}} myminio/ testkey --expiry-duration 24h
`,
}

func mainIDPOpenIDAccesskeyEdit(ctx *cli.Context) error {
	return commonAccesskeyEdit(ctx)
}
