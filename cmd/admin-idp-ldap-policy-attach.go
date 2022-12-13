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

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPLdapPolicyAttachFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user, u",
		Usage: "attach policy to user by DN or by login name",
	},
	cli.StringFlag{
		Name:  "group, g",
		Usage: "attach policy to LDAP Group DN",
	},
}

var adminIDPLdapPolicyAttachCmd = cli.Command{
	Name:         "attach",
	Usage:        "attach a policy to an entity",
	Action:       mainAdminIDPLdapPolicyAttach,
	Before:       setGlobalsFromContext,
	Flags:        append(adminIDPLdapPolicyAttachFlags, globalFlags...),
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

func mainAdminIDPLdapPolicyAttach(ctx *cli.Context) error {
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

	if err := req.IsValid(); err != nil {
		fatalIf(probe.NewError(err), "Invalid policy attach arguments.")
	}

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
