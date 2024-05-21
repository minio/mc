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

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
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
		Usage: "list access keys for the authenticated user if admin",
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
  1. Get list of all users and associated access keys in local server (if admin)
 	 {{.Prompt}} {{.HelpName}} local/

  2. Get list of users in local server (if admin)
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

type ldapUsersList struct {
	Status          string                      `json:"status"`
	DN              string                      `json:"dn"`
	STSKeys         []madmin.ServiceAccountInfo `json:"stsKeys"`
	ServiceAccounts []madmin.ServiceAccountInfo `json:"svcaccs"`
}

func (m ldapUsersList) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	o := strings.Builder{}

	o.WriteString(iFmt(0, "%s\n", labelStyle.Render("DN "+m.DN)))
	if len(m.STSKeys) > 0 {
		o.WriteString(iFmt(2, "%s\n", labelStyle.Render("STS Access Keys:")))
		for _, k := range m.STSKeys {
			o.WriteString(iFmt(4, "%s\n", k.AccessKey))
		}
	}
	if len(m.ServiceAccounts) > 0 {
		o.WriteString(iFmt(2, "%s\n", labelStyle.Render("Service Account Access Keys:")))
		for _, k := range m.ServiceAccounts {
			o.WriteString(iFmt(4, "%s\n", k.AccessKey))
		}
	}
	o.WriteString("\n")

	return o.String()
}

func (m ldapUsersList) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainIDPLdapAccesskeyList(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	usersOnly := ctx.Bool("users-only")
	tempOnly := ctx.Bool("sts-only")
	permanentOnly := ctx.Bool("svcacc-only")
	isSelf := ctx.Bool("self")
	listType := ""

	if (usersOnly && permanentOnly) || (usersOnly && tempOnly) || (permanentOnly && tempOnly) {
		e := errors.New("only one of --users-only, --temp-only, or --permanent-only can be specified")
		fatalIf(probe.NewError(e), "Invalid flags.")
	}
	if tempOnly {
		listType = "sts-only"
	} else if permanentOnly {
		listType = "svcacc-only"
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	userArg := args.Tail()

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	accessKeysMap, e := client.ListAccessKeysLDAP(globalContext, userArg, listType, isSelf)
	if e != nil {
		if e.Error() == "Access Denied." && !isSelf {
			// retry with self
			accessKeysMap, e = client.ListAccessKeysLDAP(globalContext, userArg, listType, true)
		}
		fatalIf(probe.NewError(e), "Unable to list access keys.")
	}

	for dn, accessKeys := range accessKeysMap {
		m := ldapUsersList{
			Status:          "success",
			DN:              dn,
			ServiceAccounts: accessKeys.ServiceAccounts,
			STSKeys:         accessKeys.STSKeys,
		}
		printMsg(m)
	}
	return nil
}
