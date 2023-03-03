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
	"errors"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPLdapPolicyDetachFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user, u",
		Usage: "attach policy to user by DN or by login name",
	},
	cli.StringFlag{
		Name:  "group, g",
		Usage: "attach policy to LDAP Group DN",
	},
}

var adminIDPLdapPolicyDetachCmd = cli.Command{
	Name:         "detach",
	Usage:        "detach a policy from an entity",
	Action:       mainAdminIDPLdapPolicyDetach,
	Before:       setGlobalsFromContext,
	Flags:        append(adminIDPLdapPolicyDetachFlags, globalFlags...),
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

func mainAdminIDPLdapPolicyDetach(ctx *cli.Context) error {
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
