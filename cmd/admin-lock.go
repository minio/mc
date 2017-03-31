/*
 * Minio Client (C) 2016 Minio, Inc.
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

var (
	adminLockFlags = []cli.Flag{}
)

var adminLockCmd = cli.Command{
	Name:   "lock",
	Usage:  "Control locks in servers",
	Action: mainAdminLock,
	Before: setGlobalsFromContext,
	Flags:  append(adminLockFlags, globalFlags...),
	Subcommands: []cli.Command{
		adminLockListCmd,
		adminLockClearCmd,
	},
	HideHelpCommand: true,
}

// mainAdminLock is the handle for "mc admin lock" command.
func mainAdminLock(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "list", "unlock" have their own main.
}
