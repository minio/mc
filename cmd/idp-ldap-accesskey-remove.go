// Copyright (c) 2015-2023 MinIO, Inc.
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
	"github.com/minio/mc/pkg/probe"
)

var idpLdapAccesskeyRemoveCmd = cli.Command{
	Name:         "remove",
	ShortName:    "rm",
	Usage:        "delete access key pairs for LDAP",
	Action:       mainIDPLdapAccesskeyRemove,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET ACCESSKEY

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove the access key "testkey" from local server
	 {{.Prompt}} {{.HelpName}} local/ testkey
	`,
}

func mainIDPLdapAccesskeyRemove(ctx *cli.Context) error {
	return commonAccesskeyRemove(ctx)
}

// No difference between ldap and builtin accesskey remove for now
func commonAccesskeyRemove(ctx *cli.Context) error {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	accessKey := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.DeleteServiceAccount(globalContext, accessKey)
	fatalIf(probe.NewError(e), "Unable to remove service account.")

	m := accesskeyMessage{
		op:        "remove",
		Status:    "success",
		AccessKey: accessKey,
	}

	printMsg(m)

	return nil
}
