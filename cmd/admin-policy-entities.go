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
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var adminPolicyEntitiesFlags = []cli.Flag{
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

var adminPolicyEntitiesCmd = cli.Command{
	Name:         "entities",
	Usage:        "list policy association entities",
	Action:       mainAdminPolicyEntities,
	Before:       setGlobalsFromContext,
	Flags:        append(adminPolicyEntitiesFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
  
USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all entities associated with all policies
     {{.Prompt}} {{.HelpName}} play/
  2. List all entities associated with the policies 'finteam-policy' and 'mlteam-policy'
     {{.Prompt}} {{.HelpName}} play/ --policy finteam-policy --policy mlteam-policy
  3. List all policies associated with a pair of user entities
     {{.Prompt}} {{.HelpName}} play/ --user bob --user james
  4. List all policies associated with a pair of group entities
     {{.Prompt}} {{.HelpName}} play/ --group auditors --group accounting
  5. List all entities associated with a policy, group and user
     {{.Prompt}} {{.HelpName}} play/ \
              --policy finteam-policy --user bobfisher --group consulting
`,
}

// mainAdminPolicyEntities is the handler for "mc admin policy entities" command.
func mainAdminPolicyEntities(ctx *cli.Context) error {
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

	res, e := client.GetPolicyEntities(globalContext,
		madmin.PolicyEntitiesQuery{
			Users:  usersToQuery,
			Groups: groupsToQuery,
			Policy: policiesToQuery,
		})
	fatalIf(probe.NewError(e), "Unable to fetch policy entities")

	printMsg(policyEntitiesFrom(res))
	return nil
}
