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
		Name:  "users, u",
		Usage: "only list user DNs",
	},
	cli.BoolFlag{
		Name:  "temp-only, t",
		Usage: "only list temporary access keys",
	},
	cli.BoolFlag{
		Name:  "permanent-only, p",
		Usage: "only list permanent access keys/service accounts",
	},
	cli.BoolFlag{
		Name:  "self",
		Usage: "only list access keys for the current user (only necessary if current user is admin)",
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
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get list of all users and associated access keys in local server (if admin).
 	 {{.Prompt}} {{.HelpName}} local/
  2. Get list of users in local server (if admin).
 	 {{.Prompt}} {{.HelpName}} local/ --users
  3. Get list of all users and associated temporary access keys in local server (if admin).
	 {{.Prompt}} {{.HelpName}} local/ --temp-only
  4. Get authenticated user and associated access keys in local server (if not admin).
	 {{.Prompt}} {{.HelpName}} local/
  5. Get authenticated user and associated access keys in local server (if admin).
	 {{.Prompt}} {{.HelpName}} local/ --self
	`,
}

type ldapUsersList struct {
	Status string               `json:"status"`
	Result []ldapUserAccessKeys `json:"result"`
}

type ldapUserAccessKeys struct {
	DN                  string                      `json:"dn"`
	TempAccessKeys      []madmin.ServiceAccountInfo `json:"tempAccessKeys,omitempty"`
	PermanentAccessKeys []madmin.ServiceAccountInfo `json:"permanentAccessKeys,omitempty"`
}

func (m ldapUsersList) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	o := strings.Builder{}

	for _, u := range m.Result {
		o.WriteString(iFmt(0, "%s\n", labelStyle.Render("DN "+u.DN)))
		if len(u.TempAccessKeys) > 0 {
			o.WriteString(iFmt(2, "%s\n", labelStyle.Render("Temporary Access Keys:")))
			for _, k := range u.TempAccessKeys {
				o.WriteString(iFmt(4, "%s\n", k.AccessKey))
			}
		}
		if len(u.PermanentAccessKeys) > 0 {
			o.WriteString(iFmt(2, "%s\n", labelStyle.Render("Permanent Access Keys:")))
			for _, k := range u.PermanentAccessKeys {
				o.WriteString(iFmt(4, "%s\n", k.AccessKey))
			}
		}
		o.WriteString("\n")
	}

	return o.String()
}

func (m ldapUsersList) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainIDPLdapAccesskeyList(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	usersOnly := ctx.Bool("users")
	tempOnly := ctx.Bool("temp-only")
	permanentOnly := ctx.Bool("permanent-only")
	self := ctx.Bool("self")

	if (usersOnly && permanentOnly) || (usersOnly && tempOnly) || (permanentOnly && tempOnly) {
		e := errors.New("only one of --users, --temp-only, or --permanent-only can be specified")
		fatalIf(probe.NewError(e), "Invalid flags.")
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var e error
	var users map[string]madmin.UserInfo
	if !self {
		// Assume admin access, change to user if ListUsers fails
		users, e = client.ListUsers(globalContext)
	}
	if self || e != nil {
		if self || e.Error() == "Access Denied." {
			// If user does not have ListUsers permission, or self is specified, only get current user's access keys
			users = make(map[string]madmin.UserInfo)
			users[""] = madmin.UserInfo{}
		} else {
			fatalIf(probe.NewError(e), "Unable to retrieve users.")
		}
	}

	var accessKeyList []ldapUserAccessKeys

	for dn := range users {
		if !usersOnly {
			accessKeys, _ := client.ListServiceAccounts(globalContext, dn)

			var tempAccessKeys []madmin.ServiceAccountInfo
			var permanentAccessKeys []madmin.ServiceAccountInfo

			for _, accessKey := range accessKeys.Accounts {
				if accessKey.Expiration.Unix() == 0 {
					permanentAccessKeys = append(permanentAccessKeys, accessKey)
				} else {
					tempAccessKeys = append(tempAccessKeys, accessKey)
				}
			}

			// if dn is blank, it means we are listing the current user's access keys
			if dn == "" {
				name, e := client.AccountInfo(globalContext, madmin.AccountOpts{})
				fatalIf(probe.NewError(e), "Unable to retrieve account name.")
				dn = name.AccountName
			}

			userAccessKeys := ldapUserAccessKeys{
				DN: dn,
			}
			if !tempOnly {
				userAccessKeys.PermanentAccessKeys = permanentAccessKeys
			}
			if !permanentOnly {
				userAccessKeys.TempAccessKeys = tempAccessKeys
			}

			accessKeyList = append(accessKeyList, userAccessKeys)
		} else {
			// if dn is blank, it means we are listing the current user's access keys
			if dn == "" {
				name, e := client.AccountInfo(globalContext, madmin.AccountOpts{})
				fatalIf(probe.NewError(e), "Unable to retrieve account name.")
				dn = name.AccountName
			}

			accessKeyList = append(accessKeyList, ldapUserAccessKeys{
				DN: dn,
			})
		}
	}

	m := ldapUsersList{
		Status: "success",
		Result: accessKeyList,
	}

	printMsg(m)

	return nil
}
