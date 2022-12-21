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
)

var adminDetachPolicyFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user, u",
		Usage: "detach policy from user",
	},
	cli.StringFlag{
		Name:  "group, g",
		Usage: "detach policy from group",
	},
}

var adminPolicyDetachCmd = cli.Command{
	Name:         "detach",
	Usage:        "detach an IAM policy from a user or group",
	Action:       mainAdminPolicyDetach,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminDetachPolicyFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET POLICY [POLICY...] [--user USER | --group GROUP]

  Exactly one of --user or --group is required.

POLICY:
  Name of the policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Detach the "readonly" policy from user "james".
     {{.Prompt}} {{.HelpName}} myminio readonly --user james
  2. Detach the "audit-policy" and "acct-policy" policies from group "legal".
     {{.Prompt}} {{.HelpName}} myminio audit-policy acct-policy --group legal
`,
}

// mainAdmihPolicyDetach is the handler for "mc admin policy detach" command.
func mainAdminPolicyDetach(ctx *cli.Context) error {
	return userAttachOrDetachPolicy(ctx, false)
}
