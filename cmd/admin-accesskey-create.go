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

var adminAccesskeyCreateFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "access-key",
		Usage: "set an access key for the account",
	},
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

var adminAccesskeyCreateCmd = cli.Command{
	Name:         "create",
	Usage:        "create access key pairs for users",
	Action:       mainAdminAccesskeyCreate,
	Before:       setGlobalsFromContext,
	Flags:        append(adminAccesskeyCreateFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a new access key pair with the same policy as the authenticated user
     {{.Prompt}} {{.HelpName}} myminio/

  2. Create a new access key pair with custom access key and secret key
     {{.Prompt}} {{.HelpName}} myminio/ --access-key myaccesskey --secret-key mysecretkey

  3. Create a new access key pair for user 'tester' that expires in 1 day
     {{.Prompt}} {{.HelpName}} myminio/ tester --expiry-duration 24h

  4. Create a new access key pair for authenticated user that expires on 2025-01-01
     {{.Prompt}} {{.HelpName}} --expiry 2025-01-01

  5. Create a new access key pair for user 'tester' with a custom policy
	 {{.Prompt}} {{.HelpName}} myminio/ tester --policy /path/to/policy.json

  6. Create a new access key pair for user 'tester' with a custom name and description
	 {{.Prompt}} {{.HelpName}} myminio/ tester --name "Tester's Access Key" --description "Access key for tester"
`,
}

func mainAdminAccesskeyCreate(ctx *cli.Context) error {
	return commonAccesskeyCreate(ctx, false)
}
