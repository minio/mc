// Copyright (c) 2015-2021 MinIO, Inc.
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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/pkg/console"
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
	OnUsageError:    onUsageError,
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
