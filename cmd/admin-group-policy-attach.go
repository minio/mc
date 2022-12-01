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

var adminGroupPolicyAttachCmd = cli.Command{
	Name:         "attach",
	Usage:        "attach an IAM policy to a group",
	Action:       mainAdminGroupPolicyAttach,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET GROUPNAME POLICYNAME...

POLICYNAME:
  Name of the policy on the MinIO server, may be multiple policies.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Attach the "diagnostics" policy to the group "auditors".
     {{.Prompt}} {{.HelpName}} myminio diagnostics auditors

  2. Add user "james" to group "staff", then add the "readwrite" policy to the group "staff".
     {{.Prompt}} mc admin group add myminio staff james
     {{.Prompt}} {{.HelpName}} myminio staff readwrite
`,
}

// mainAdminGroupPolicyAttach is the handler for "mc admin group policy attach" command.
func mainAdminGroupPolicyAttach(ctx *cli.Context) error {
	return groupAttachOrDetachPolicy(ctx, true)
}
