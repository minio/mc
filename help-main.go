/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

package main

import "github.com/minio/cli"

// Help command.
var helpCmd = cli.Command{
	Name:   "help",
	Usage:  "Show help.",
	Action: mainHelp,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [COMMAND]

EXAMPLES:
   1. Show mc usage, commands and global flags.
     $ mc {{.Name}}

   2. Show help for a command.
      $ mc {{.Name}} share
`,
}

// Validate command line arguments.
func checkHelpSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 1 { // Accepts a maximum of 1 argument.
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "help", exitCode)
	}
}

// main for help command.
func mainHelp(ctx *cli.Context) {
	checkHelpSyntax(ctx)

	if ctx.Args().First() != "" { // command help.
		cli.ShowCommandHelp(ctx, ctx.Args().First())
	} else { // mc help.
		cli.ShowAppHelp(ctx)
	}
}
