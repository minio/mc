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

var adminServiceCmd = cli.Command{
	Name:   "service",
	Usage:  "Control servers.",
	Action: mainAdminService,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		adminServiceRestartCmd,
		adminServiceStatusCmd,
	},
	CustomHelpTemplate: `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} [FLAGS] COMMAND

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
COMMANDS:
   {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}
`,
}

// mainAdmin is the handle for "mc admin service" command.
func mainAdminService(ctx *cli.Context) error {

	if ctx.Args().First() != "" { // command help.
		cli.ShowCommandHelp(ctx, ctx.Args().First())
	} else {
		// command with Subcommands is an App.
		cli.ShowAppHelp(ctx)
	}

	return nil
	// Sub-commands like "status", "stop", "restart" have their own main.
}
