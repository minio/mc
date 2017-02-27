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

package cmd

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
	configFlags = []cli.Flag{}
)

var configCmd = cli.Command{
	Name:            "config",
	Usage:           "Manage mc configuration file.",
	Action:          mainConfig,
	Before:          setGlobalsFromContext,
	HideHelpCommand: true,
	Flags:           append(configFlags, globalFlags...),
	Subcommands: []cli.Command{
		configHostCmd,
	},
}

// mainConfig is the handle for "mc config" command. provides sub-commands which write configuration data in json format to config file.
func mainConfig(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "host" and "alias" have their own main.
}
