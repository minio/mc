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
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPLdapPolicyEntitiesFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "user, u",
		Usage: "list policies associated with user(s)",
	},
	cli.StringSliceFlag{
		Name:  "group, g",
		Usage: "list policies associated with group(s)",
	},
	cli.StringSliceFlag{
		Name:  "policy, p",
		Usage: "list users or groups associated with policy",
	},
}

var adminIDPLdapPolicyEntitiesCmd = cli.Command{
	Name:         "entities",
	Usage:        "list policy association entities",
	Action:       mainAdminIDPLdapPolicyEntities,
	Before:       setGlobalsFromContext,
	Flags:        append(adminIDPLdapPolicyEntitiesFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all LDAP entities associated with all policies
     {{.Prompt}} {{.HelpName}} play/
  2. List all LDAP entities associated with the policies 'finteam-policy' and 'mlteam-policy'
     {{.Prompt}} {{.HelpName}} play/ --policy finteam-policy --policy mlteam-policy
  3. List all policies associated with a pair of User LDAP entities
     {{.Prompt}} {{.HelpName}} play/ \
              --user 'uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io' \
              --user 'uid=fahim,ou=people,ou=swengg,dc=min,dc=io'
  4. List all policies associated with a pair of Group LDAP entities
     {{.Prompt}} {{.HelpName}} play/ \
              --group 'cn=projecta,ou=groups,ou=swengg,dc=min,dc=io' \
              --group 'cn=projectb,ou=groups,ou=swengg,dc=min,dc=io'
  5. List all entities associated with a policy, group and user
     {{.Prompt}} {{.HelpName}} play/ \
              --policy finteam-policy
              --user 'uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io' \
              --group 'cn=projectb,ou=groups,ou=swengg,dc=min,dc=io'
`,
}

func mainAdminIDPLdapPolicyEntities(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1)
	}

	usersToQuery := ctx.StringSlice("user")
	groupsToQuery := ctx.StringSlice("group")
	policiesToQuery := ctx.StringSlice("policy")

	args := ctx.Args()

	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	res, e := client.GetLDAPPolicyEntities(globalContext,
		madmin.PolicyEntitiesQuery{
			Users:  usersToQuery,
			Groups: groupsToQuery,
			Policy: policiesToQuery,
		})
	fatalIf(probe.NewError(e), "Unable to fetch LDAP policy entities")

	printMsg(policyEntitiesFrom(res))
	return nil
}

type policyEntities struct {
	Status string                      `json:"status"`
	Result madmin.PolicyEntitiesResult `json:"result"`
}

func policyEntitiesFrom(r madmin.PolicyEntitiesResult) policyEntities {
	return policyEntities{
		Status: "success",
		Result: r,
	}
}

func (p policyEntities) JSON() string {
	bs, e := json.MarshalIndent(p, "", "  ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(bs)
}

func iFmt(n int, fmtStr string, a ...any) string {
	indentStr := ""
	if n > 0 {
		s := make([]rune, n)
		for i := range s {
			s[i] = ' '
		}
		indentStr = string(s)
	}
	return fmt.Sprintf(indentStr+fmtStr, a...)
}

func (p policyEntities) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")) // green
	o := strings.Builder{}

	o.WriteString(iFmt(0, "%s %s\n",
		labelStyle.Render("Query time:"),
		p.Result.Timestamp.Format(time.RFC3339)))

	if len(p.Result.UserMappings) > 0 {
		o.WriteString(iFmt(0, "%s\n", labelStyle.Render("User -> Policy Mappings:")))

		for _, u := range p.Result.UserMappings {
			o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("User:"), u.User))
			for _, p := range u.Policies {
				o.WriteString(iFmt(4, "%s\n", p))
			}
		}
	}
	if len(p.Result.GroupMappings) > 0 {
		o.WriteString(iFmt(0, "%s\n", labelStyle.Render("Group -> Policy Mappings:")))

		for _, u := range p.Result.GroupMappings {
			o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("Group:"), u.Group))
			for _, p := range u.Policies {
				o.WriteString(iFmt(4, "%s\n", p))
			}
		}
	}
	if len(p.Result.PolicyMappings) > 0 {
		o.WriteString(iFmt(0, "%s\n", labelStyle.Render("Policy -> Entity Mappings:")))

		for _, u := range p.Result.PolicyMappings {
			o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("Policy:"), u.Policy))
			if len(u.Users) > 0 {
				o.WriteString(iFmt(4, "%s\n", labelStyle.Render("User Mappings:")))
				for _, p := range u.Users {
					o.WriteString(iFmt(6, "%s\n", p))
				}
			}
			if len(u.Groups) > 0 {
				o.WriteString(iFmt(4, "%s\n", labelStyle.Render("Group Mappings:")))
				for _, p := range u.Groups {
					o.WriteString(iFmt(6, "%s\n", p))
				}
			}
		}
	}

	return o.String()
}
