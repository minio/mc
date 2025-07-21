// Copyright (c) 2015-2025 MinIO, Inc.
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

var idpOpenIDAccesskeyListFlags = []cli.Flag{
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
		Usage: "list all access keys for all OpenID users",
	},
	cli.BoolFlag{
		Name:  "all-configs",
		Usage: "list access keys for all OpenID configurations",
	},
}

var idpOpenidAccesskeyListCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "list access key pairs for OpenID",
	Action:       mainIDPOpenIDAccesskeyList,
	Before:       setGlobalsFromContext,
	Flags:        append(idpOpenIDAccesskeyListFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET[:CFGNAME] [USER/ID...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get list of all OpenID users and associated access keys in local server (if admin)
 	 {{.Prompt}} {{.HelpName}} local/

  2. Get list of OpenID users in local server (if admin)
 	 {{.Prompt}} {{.HelpName}} local/ --users-only

  3. Get list of all users and associated temporary access keys in play server (if admin)
	 {{.Prompt}} {{.HelpName}} play/ --temp-only

  4. Get list of access keys associated with internal name 'openidinternalname'
  	 {{.Prompt}} {{.HelpName}} play/ openidinternalname

  5. Get list of access keys associated with ID claim 'openidsub' (default claim is sub)
	 {{.Prompt}} {{.HelpName}} play/ openidsub

  7. Get authenticated user and associated access keys in local server (if not admin)
	 {{.Prompt}} {{.HelpName}} local/
`,
}

type openIDAccesskeyList struct {
	Status     string                        `json:"status"`
	ConfigName string                        `json:"configName"`
	Users      []madmin.OpenIDUserAccessKeys `json:"users"`
}

func (m openIDAccesskeyList) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	o := strings.Builder{}

	o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Config Name:"), m.ConfigName))
	userStr := "User ID"
	for _, user := range m.Users {
		o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render(userStr+":"), user.MinioAccessKey))
		o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("ID:"), user.ID))
		if user.ReadableName != "" {
			o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("Readable Name:"), user.ReadableName))
		}
		if len(user.STSKeys) > 0 || len(user.ServiceAccounts) > 0 {
			o.WriteString(iFmt(4, "%s\n", labelStyle.Render("Access Keys:")))
		}
		for _, k := range user.STSKeys {
			expiration := "never"
			if nilExpiry(k.Expiration) != nil {
				expiration = humanize.Time(*k.Expiration)
			}
			o.WriteString(iFmt(6, "%s, expires: %s, sts: true\n", k.AccessKey, expiration))
		}
		for _, k := range user.ServiceAccounts {
			expiration := "never"
			if nilExpiry(k.Expiration) != nil {
				expiration = humanize.Time(*k.Expiration)
			}
			o.WriteString(iFmt(6, "%s, expires: %s, sts: false\n", k.AccessKey, expiration))
		}
	}

	return o.String()
}

func (m openIDAccesskeyList) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainIDPOpenIDAccesskeyList(ctx *cli.Context) error {
	aliasedURL, tentativeAll, users, opts := commonAccesskeyList(ctx)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	accessKeys, e := client.ListAccessKeysOpenIDBulk(globalContext, users, opts)
	if e != nil {
		if e.Error() == "Access Denied." && tentativeAll {
			// retry with self
			opts.All = false
			accessKeys, e = client.ListAccessKeysOpenIDBulk(globalContext, users, opts)
		}
		fatalIf(probe.NewError(e), "Unable to list access keys.")
	}

	for _, cfg := range accessKeys {
		m := openIDAccesskeyList{
			Status:     "success",
			ConfigName: cfg.ConfigName,
			Users:      cfg.Users,
		}
		printMsg(m)
	}
	return nil
}
