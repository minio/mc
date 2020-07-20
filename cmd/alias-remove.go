/*
 * MinIO Client (C) 2017-2020 MinIO, Inc.
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

import (
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

var aliasRemoveCmd = cli.Command{
	Name:      "remove",
	ShortName: "rm",
	Usage:     "remove an alias from configuration file",
	Action: func(ctx *cli.Context) error {
		return mainAliasRemove(ctx, false)
	},
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove "goodisk" alias from the configuration.
     {{.Prompt}} {{.HelpName}} goodisk

`,
}

// checkAliasRemoveSyntax - verifies input arguments to 'alias remove'.
func checkAliasRemoveSyntax(ctx *cli.Context) {
	args := ctx.Args()

	if len(ctx.Args()) != 1 {
		fatalIf(errInvalidArgument().Trace(args...),
			"Incorrect number of arguments for alias remove command.")
	}

	alias := cleanAlias(args.Get(0))
	if !isValidAlias(alias) {
		fatalIf(errDummy().Trace(alias), "Invalid alias `"+alias+"`.")
	}
}

// mainAliasRemove is the handle for "mc alias rm" command.
func mainAliasRemove(ctx *cli.Context, deprecated bool) error {
	checkAliasRemoveSyntax(ctx)

	console.SetColor("AliasMessage", color.New(color.FgGreen))

	args := ctx.Args()
	alias := args.Get(0)

	aliasMsg := removeAlias(alias) // Remove an alias
	aliasMsg.op = "remove"
	printMsg(aliasMsg)
	return nil
}

// removeAlias - removes an alias.
func removeAlias(alias string) aliasMessage {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version `"+globalMCConfigVersion+"`.")

	// Remove the alias from the config.
	delete(conf.Aliases, alias)

	err = saveMcConfig(conf)
	fatalIf(err.Trace(alias), "Unable to save the delete alias in config version `"+globalMCConfigVersion+"`.")

	return aliasMessage{Alias: alias}
}
