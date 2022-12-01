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

var adminUserPolicyDetachCmd = cli.Command{
	Name:         "detach",
	Usage:        "detach an IAM policy from a user",
	Action:       mainAdminUserPolicyDetach,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET USERNAME POLICYNAME [POLICYNAME...]

POLICYNAME:
  Name of the policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Detach the "diagnostics" policy from user "james".
     {{.Prompt}} {{.HelpName}} myminio diagnostics james
`,
}

// mainAdminUserPolicyDetach is the handler for "mc admin group policy detach" command.
func mainAdminUserPolicyDetach(ctx *cli.Context) error {
	return userAttachOrDetachPolicy(ctx, false)
}
