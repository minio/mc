/*
 * MinIO Client (C) 2017 MinIO, Inc.
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

var configHostRemoveCmd = cli.Command{
	Name:            "remove",
	ShortName:       "rm",
	Usage:           "remove a host from configuration file",
	Action:          mainConfigHostRemove,
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
  1. Remove "goodisk" from config.
     {{.Prompt}} {{.HelpName}} goodisk

`,
}

// checkConfigHostRemoveSyntax - verifies input arguments to 'config host remove'.
func checkConfigHostRemoveSyntax(ctx *cli.Context) {
	args := ctx.Args()

	if len(ctx.Args()) != 1 {
		fatalIf(errInvalidArgument().Trace(args...),
			"Incorrect number of arguments for remove host command.")
	}

	alias := cleanAlias(args.Get(0))
	if !isValidAlias(alias) {
		fatalIf(errDummy().Trace(alias), "Invalid alias `"+alias+"`.")
	}
}

// mainConfigHost is the handle for "mc config host rm" command.
func mainConfigHostRemove(ctx *cli.Context) error {
	checkConfigHostRemoveSyntax(ctx)

	console.SetColor("HostMessage", color.New(color.FgGreen))

	args := ctx.Args()
	alias := args.Get(0)
	removeHost(alias) // Remove a host.
	return nil
}

// removeHost - removes a host.
func removeHost(alias string) {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version `"+globalMCConfigVersion+"`.")

	// Remove host.
	delete(conf.Hosts, alias)

	err = saveMcConfig(conf)
	fatalIf(err.Trace(alias), "Unable to save deleted hosts in config version `"+globalMCConfigVersion+"`.")

	printMsg(hostMessage{op: "remove", Alias: alias})
}
