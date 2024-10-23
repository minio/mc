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
	"strings"

	"github.com/charmbracelet/lipgloss"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var adminAccesskeyListFlags = []cli.Flag{
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
		Usage: "list all access keys for all builtin users",
	},
}

var adminAccesskeyListCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "list access key pairs for builtin users",
	Action:       mainAdminAccesskeyList,
	Before:       setGlobalsFromContext,
	Flags:        append(adminAccesskeyListFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [DN...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get list of all builtin users and associated access keys in local server
	 {{.Prompt}} {{.HelpName}} local/ --all

  2. Get list of access keys for the authenticated user in local server
	 {{.Prompt}} {{.HelpName}} local/ --self

  3. Get list of builtin users in local server
	 {{.Prompt}} {{.HelpName}} local/ --all --users-only

  4. Get list of all builtin users and associated temporary access keys in play server (if admin)
	 {{.Prompt}} {{.HelpName}} play/ --temp-only

  5. Get list of access keys associated with user 'foobar'
	 {{.Prompt}} {{.HelpName}} play/ foobar

  6. Get list of access keys associated with users 'foobar' and 'tester'
	 {{.Prompt}} {{.HelpName}} play/ foobar tester

  7. Get all users and access keys if admin, else get authenticated user and associated access keys
	 {{.Prompt}} {{.HelpName}} local/
`,
}

type userAccesskeyList struct {
	Status          string                      `json:"status"`
	User            string                      `json:"user"`
	STSKeys         []madmin.ServiceAccountInfo `json:"stsKeys"`
	ServiceAccounts []madmin.ServiceAccountInfo `json:"svcaccs"`
	LDAP            bool                        `json:"ldap,omitempty"`
}

func (m userAccesskeyList) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	o := strings.Builder{}

	userStr := "User"
	if m.LDAP {
		userStr = "DN"
	}
	o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render(userStr+":"), m.User))
	if len(m.STSKeys) > 0 || len(m.ServiceAccounts) > 0 {
		o.WriteString(iFmt(2, "%s\n", labelStyle.Render("Access Keys:")))
	}
	for _, k := range m.STSKeys {
		expiration := "never"
		if nilExpiry(k.Expiration) != nil {
			expiration = humanize.Time(*k.Expiration)
		}
		o.WriteString(iFmt(4, "%s, expires: %s, sts: true\n", k.AccessKey, expiration))
	}
	for _, k := range m.ServiceAccounts {
		expiration := "never"
		if nilExpiry(k.Expiration) != nil {
			expiration = humanize.Time(*k.Expiration)
		}
		o.WriteString(iFmt(4, "%s, expires: %s, sts: false\n", k.AccessKey, expiration))
	}

	return o.String()
}

func (m userAccesskeyList) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainAdminAccesskeyList(ctx *cli.Context) error {
	aliasedURL, tentativeAll, users, opts := commonAccesskeyList(ctx)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	accessKeysMap, e := client.ListAccessKeysBulk(globalContext, users, opts)
	if e != nil {
		if e.Error() == "Access Denied." && tentativeAll {
			// retry with self
			opts.All = false
			accessKeysMap, e = client.ListAccessKeysBulk(globalContext, users, opts)
		}
		fatalIf(probe.NewError(e), "Unable to list access keys.")
	}

	for user, accessKeys := range accessKeysMap {
		m := userAccesskeyList{
			Status:          "success",
			User:            user,
			ServiceAccounts: accessKeys.ServiceAccounts,
			STSKeys:         accessKeys.STSKeys,
			LDAP:            false,
		}
		printMsg(m)
	}
	return nil
}
