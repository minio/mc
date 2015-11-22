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

//   Configure minio client
//
//   ----
//   NOTE: that the configure command only writes values to the config file.
//   It does not use any configuration values from the environment variables.
//
//   One needs to edit configuration file manually, this is purposefully done
//   so to avoid taking credentials over cli arguments. It is a security precaution
//   ----
//

var (
	configFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of config.",
	}
)

var configCmd = cli.Command{
	Name:   "config",
	Usage:  "Manage configuration file.",
	Action: mainConfig,
	Flags:  append(globalFlags, configFlagHelp),
	Subcommands: []cli.Command{
		configAliasCmd,
		configHostCmd,
		configVersionCmd,
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

// mainConfig is the handle for "mc config" command. provides sub-commands which write configuration data in json format to config file.
func mainConfig(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	if ctx.Args().First() != "" { // command help.
		cli.ShowCommandHelp(ctx, ctx.Args().First())
	} else {
		// command with Subcommands is an App.
		cli.ShowAppHelp(ctx)
	}

	// Sub-commands like "host" and "alias" have their own main.
}
