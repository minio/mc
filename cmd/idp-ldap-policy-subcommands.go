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
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
)

var idpLdapPolicyAttachFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user, u",
		Usage: "attach policy to user by DN or by login name",
	},
	cli.StringFlag{
		Name:  "group, g",
		Usage: "attach policy to LDAP Group DN",
	},
}

var idpLdapPolicyAttachCmd = cli.Command{
	Name:         "attach",
	Usage:        "attach a policy to an entity",
	Action:       mainIDPLdapPolicyAttach,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapPolicyAttachFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET POLICY [POLICY...] [ --user=USER | --group=GROUP ]

  Exactly one "--user" or "--group" flag is required.

POLICY:
  Name of a policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Attach policy "mypolicy" to a user
     {{.Prompt}} {{.HelpName}} play/ mypolicy --user='uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io'
  2. Attach policies "policy1" and "policy2" to a group
     {{.Prompt}} {{.HelpName}} play/ policy1 policy2 --group='cn=projectb,ou=groups,ou=swengg,dc=min,dc=io'
`,
}

// Quote from AWS policy naming requirement (ref:
// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_iam-quotas.html):
//
// Names of users, groups, roles, policies, instance profiles, and server
// certificates must be alphanumeric, including the following common characters:
// plus (+), equal (=), comma (,), period (.), at (@), underscore (_), and
// hyphen (-).

func mainIDPLdapPolicyAttach(ctx *cli.Context) error {
	// We need exactly one alias, and at least one policy.
	if len(ctx.Args()) < 2 {
		showCommandHelpAndExit(ctx, 1)
	}
	user := ctx.String("user")
	group := ctx.String("group")

	args := ctx.Args()
	aliasedURL := args.Get(0)

	policies := args[1:]
	req := madmin.PolicyAssociationReq{
		Policies: policies,
		User:     user,
		Group:    group,
	}
	fatalIf(probe.NewError(req.IsValid()), "Invalid policy attach arguments.")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	res, e := client.AttachPolicyLDAP(globalContext, req)
	fatalIf(probe.NewError(e), "Unable to make LDAP policy association")

	m := policyAssociationMessage{
		attach:           true,
		Status:           "success",
		PoliciesAttached: res.PoliciesAttached,
		User:             user,
		Group:            group,
	}
	printMsg(m)
	return nil
}

type policyAssociationMessage struct {
	attach           bool
	Status           string   `json:"status"`
	PoliciesAttached []string `json:"policiesAttached,omitempty"`
	PoliciesDetached []string `json:"policiesDetached,omitempty"`
	User             string   `json:"user,omitempty"`
	Group            string   `json:"group,omitempty"`
}

func (m policyAssociationMessage) String() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")) // green

	policiesS := style.Render("Attached Policies:")
	entityS := style.Render("To User:")
	policies := m.PoliciesAttached
	entity := m.User
	switch {
	case m.User != "" && m.attach:
	case m.User != "" && !m.attach:
		policiesS = style.Render("Detached Policies:")
		policies = m.PoliciesDetached
		entityS = style.Render("From User:")
	case m.Group != "" && m.attach:
		entityS = style.Render("To Group:")
		entity = m.Group
	case m.Group != "" && !m.attach:
		policiesS = style.Render("Detached Policies:")
		policies = m.PoliciesDetached
		entityS = style.Render("From Group:")
		entity = m.Group
	}
	return fmt.Sprintf("%s %v\n%s %s\n", policiesS, policies, entityS, entity)
}

func (m policyAssociationMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

var idpLdapPolicyDetachFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user, u",
		Usage: "attach policy to user by DN or by login name",
	},
	cli.StringFlag{
		Name:  "group, g",
		Usage: "attach policy to LDAP Group DN",
	},
}

var idpLdapPolicyDetachCmd = cli.Command{
	Name:         "detach",
	Usage:        "detach a policy from an entity",
	Action:       mainIDPLdapPolicyDetach,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapPolicyDetachFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET POLICY [POLICY...] [ --user=USER | --group=GROUP ]

  Exactly one of "--user" or "--group" is required.

POLICY:
  Name of a policy on the MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Detach policy "mypolicy" from a user
     {{.Prompt}} {{.HelpName}} play/ mypolicy --user='uid=bobfisher,ou=people,ou=hwengg,dc=min,dc=io'
  2. Detach policies "policy1" and "policy2" from a group
     {{.Prompt}} {{.HelpName}} play/ policy1 policy2 --group='cn=projectb,ou=groups,ou=swengg,dc=min,dc=io'
`,
}

// Quote from AWS policy naming requirement (ref:
// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_iam-quotas.html):
//
// Names of users, groups, roles, policies, instance profiles, and server
// certificates must be alphanumeric, including the following common characters:
// plus (+), equal (=), comma (,), period (.), at (@), underscore (_), and
// hyphen (-).

func mainIDPLdapPolicyDetach(ctx *cli.Context) error {
	// We need exactly one alias, and at least one policy.
	if len(ctx.Args()) < 2 {
		showCommandHelpAndExit(ctx, 1)
	}

	user := ctx.String("user")
	group := ctx.String("group")

	if user == "" && group == "" {
		e := errors.New("at least one of --user or --group is required.")
		fatalIf(probe.NewError(e), "Missing flag in command")
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	policies := args[1:]

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	res, e := client.DetachPolicyLDAP(globalContext,
		madmin.PolicyAssociationReq{
			Policies: policies,
			User:     user,
			Group:    group,
		})
	fatalIf(probe.NewError(e), "Unable to make LDAP policy association")

	m := policyAssociationMessage{
		attach:           false,
		Status:           "success",
		PoliciesDetached: res.PoliciesDetached,
		User:             user,
		Group:            group,
	}
	printMsg(m)
	return nil
}

var idpLdapPolicyEntitiesFlags = []cli.Flag{
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

var idpLdapPolicyEntitiesCmd = cli.Command{
	Name:         "entities",
	Usage:        "list policy association entities",
	Action:       mainIDPLdapPolicyEntities,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapPolicyEntitiesFlags, globalFlags...),
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

func mainIDPLdapPolicyEntities(ctx *cli.Context) error {
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

func builderWrapper(strList []string, o *strings.Builder, indent, maxLen int) {
	currLen := 0
	for _, s := range strList {
		if currLen+len(s) > maxLen && currLen > 0 {
			o.WriteString("\n")
			currLen = 0
		}
		if currLen == 0 {
			o.WriteString(iFmt(indent, ""))
			currLen = indent
		} else {
			o.WriteString(", ")
			currLen += 2
		}
		if strings.Contains(s, ",") {
			s = fmt.Sprintf("\"%s\"", s)
		}
		o.WriteString(s)
		currLen += len(s)
	}
	o.WriteString("\n")
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
			o.WriteString(iFmt(4, "%s\n", labelStyle.Render("Policies:")))
			builderWrapper(u.Policies, &o, 6, 80)

			if len(u.MemberOfMappings) > 0 {
				effectivePolicies := set.CreateStringSet(u.Policies...)
				o.WriteString(iFmt(4, "%s\n", labelStyle.Render("Group Memberships:")))
				groups := make([]string, 0, len(u.MemberOfMappings))
				for _, g := range u.MemberOfMappings {
					groups = append(groups, g.Group)
					for _, p := range g.Policies {
						effectivePolicies.Add(p)
					}
				}
				builderWrapper(groups, &o, 6, 80)

				o.WriteString(iFmt(4, "%s\n", labelStyle.Render("Effective Policies:")))
				builderWrapper(effectivePolicies.ToSlice(), &o, 6, 80)
			}
		}
	}
	if len(p.Result.GroupMappings) > 0 {
		o.WriteString(iFmt(0, "%s\n", labelStyle.Render("Group -> Policy Mappings:")))

		for _, u := range p.Result.GroupMappings {
			o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("Group:"), u.Group))
			o.WriteString(iFmt(4, "%s\n", labelStyle.Render("Policies:")))
			for _, p := range u.Policies {
				o.WriteString(iFmt(6, "%s\n", p))
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
