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
	"errors"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var idpLdapAccesskeyListFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "users-only",
		Usage: "only list user DNs",
	},
	cli.BoolFlag{
		Name:  "temp-only",
		Usage: "only list temporary access keys",
	},
	cli.BoolFlag{
		Name:  "svcacc-only",
		Usage: "only list service account access keys",
	},
	cli.BoolFlag{
		Name:  "self",
		Usage: "list access keys for the authenticated user",
	},
	cli.BoolFlag{
		Name:  "all",
		Usage: "list all access keys for all LDAP users",
	},
}

var idpLdapAccesskeyListCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "list access key pairs for LDAP",
	Action:       mainIDPLdapAccesskeyList,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapAccesskeyListFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [DN...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get list of all LDAP users and associated access keys in local server (if admin)
 	 {{.Prompt}} {{.HelpName}} local/

  2. Get list of LDAP users in local server (if admin)
 	 {{.Prompt}} {{.HelpName}} local/ --users-only

  3. Get list of all users and associated temporary access keys in play server (if admin)
	 {{.Prompt}} {{.HelpName}} play/ --temp-only

  4. Get list of access keys associated with user 'bobfisher'
  	 {{.Prompt}} {{.HelpName}} play/ uid=bobfisher,dc=min,dc=io

  5. Get list of access keys associated with user 'bobfisher' (alt)
	 {{.Prompt}} {{.HelpName}} play/ bobfisher

  6. Get list of access keys associated with users 'bobfisher' and 'cody3'
  	 {{.Prompt}} {{.HelpName}} play/ uid=bobfisher,dc=min,dc=io uid=cody3,dc=min,dc=io

  7. Get authenticated user and associated access keys in local server (if not admin)
	 {{.Prompt}} {{.HelpName}} local/
`,
}

func mainIDPLdapAccesskeyList(ctx *cli.Context) error {
	aliasedURL, tentativeAll, users, opts := commonAccesskeyList(ctx)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	accessKeysMap, e := client.ListAccessKeysLDAPBulkWithOpts(globalContext, users, opts)
	if e != nil {
		if e.Error() == "Access Denied." && tentativeAll {
			// retry with self
			opts.All = false
			accessKeysMap, e = client.ListAccessKeysLDAPBulkWithOpts(globalContext, users, opts)
		}
		fatalIf(probe.NewError(e), "Unable to list access keys.")
	}

	for dn, accessKeys := range accessKeysMap {
		m := userAccesskeyList{
			Status:          "success",
			User:            dn,
			ServiceAccounts: accessKeys.ServiceAccounts,
			STSKeys:         accessKeys.STSKeys,
			LDAP:            true,
		}
		printMsg(m)
	}
	return nil
}

func commonAccesskeyList(ctx *cli.Context) (aliasedURL string, tentativeAll bool, users []string, opts madmin.ListAccessKeysOpts) {
	if len(ctx.Args()) == 0 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	usersOnly := ctx.Bool("users-only")
	stsOnly := ctx.Bool("temp-only")
	svcaccOnly := ctx.Bool("svcacc-only")
	selfFlag := ctx.Bool("self")
	opts.All = ctx.Bool("all")
	opts.AllConfigs = ctx.Bool("all-configs")

	args := ctx.Args()
	aliasedURL = args.Get(0)
	users = args.Tail()

	aliasedURL, cfgName, _ := strings.Cut(aliasedURL, ":")
	if cfgName != "" {
		opts.ConfigName = cfgName
	}

	var e error
	if (usersOnly && svcaccOnly) || (usersOnly && stsOnly) || (svcaccOnly && stsOnly) {
		e = errors.New("only one of --users-only, --temp-only, or --permanent-only can be specified")
	} else if selfFlag && opts.All {
		e = errors.New("only one of --self or --all can be specified")
	} else if (selfFlag || opts.All) && len(users) > 0 {
		e = errors.New("users cannot be specified with --self or --all")
	}
	fatalIf(probe.NewError(e), "Invalid flags.")

	// If no users/self/all flags are specified, tentatively assume --all
	// If access is denied on tentativeAll, retry with self
	// This is to maintain compatibility with the previous behavior
	if !selfFlag && !opts.All && len(users) == 0 {
		tentativeAll = true
		opts.All = true
	}

	switch {
	case usersOnly:
		opts.ListType = madmin.AccessKeyListUsersOnly
	case stsOnly:
		opts.ListType = madmin.AccessKeyListSTSOnly
	case svcaccOnly:
		opts.ListType = madmin.AccessKeyListSvcaccOnly
	default:
		opts.ListType = madmin.AccessKeyListAll
	}

	return aliasedURL, tentativeAll, users, opts
}
