/*
 * Minio Client (C) 2018 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import "github.com/minio/cli"

var adminPoliciesCmd = cli.Command{
	Name:   "policies",
	Usage:  "Manage canned policies",
	Action: mainAdminPolicies,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		adminPoliciesAddCmd,
		adminPoliciesRemoveCmd,
		adminPoliciesListCmd,
	},
	HideHelpCommand: true,
}

// mainAdminPolicies is the handle for "mc admin config" command.
func mainAdminPolicies(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "get", "set" have their own main.
}
