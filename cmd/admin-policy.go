/*
 * MinIO Client (C) 2018-2019 MinIO, Inc.
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

var adminPolicyCmd = cli.Command{
	Name:   "policy",
	Usage:  "manage policies defined in the MinIO server",
	Action: mainAdminPolicy,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		adminPolicyAddCmd,
		adminPolicyRemoveCmd,
		adminPolicyListCmd,
		adminPolicyInfoCmd,
		adminPolicySetCmd,
	},
	HideHelpCommand: true,
}

// mainAdminPolicy is the handle for "mc admin policy" command.
func mainAdminPolicy(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "get", "set" have their own main.
}
