// Copyright (c) 2015-2022 MinIO, Inc.
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
	"fmt"
	"sort"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/pkg/v3/console"
	"github.com/minio/pkg/v3/env"

	"github.com/fatih/color"
)

var aliasListCmd = cli.Command{
	Name:      "list",
	ShortName: "ls",
	Usage:     "list aliases in configuration file",
	Action: func(ctx *cli.Context) error {
		return mainAliasList(ctx, false)
	},
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	OnUsageError:    onUsageError,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [ALIAS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all aliases.
     {{.Prompt}} {{.HelpName}}

  2. List a specific alias.
     {{.Prompt}} {{.HelpName}} s3
`,
}

// Input argument validator..
func checkAliasListSyntax(ctx *cli.Context) {
	args := ctx.Args()

	if len(ctx.Args()) > 1 {
		fatalIf(errInvalidArgument().Trace(args...),
			"Incorrect number of arguments to alias list command.")
	}
}

func mainAliasList(ctx *cli.Context, deprecated bool) error {
	checkAliasListSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("Alias", color.New(color.FgCyan, color.Bold))
	console.SetColor("URL", color.New(color.FgYellow))
	console.SetColor("AccessKey", color.New(color.FgCyan))
	console.SetColor("SecretKey", color.New(color.FgCyan))
	console.SetColor("API", color.New(color.FgBlue))
	console.SetColor("Path", color.New(color.FgCyan))
	console.SetColor("Src", color.New(color.FgCyan))

	alias := cleanAlias(ctx.Args().Get(0))

	aliasesMsgs := listAliases(alias, deprecated) // List all configured hosts.
	for i := range aliasesMsgs {
		aliasesMsgs[i].op = "list"
	}
	printAliases(aliasesMsgs...)
	return nil
}

// Prints all the aliases
func printAliases(aliases ...aliasMessage) {
	maxAlias := 0
	for _, alias := range aliases {
		if len(alias.Alias) > maxAlias {
			maxAlias = len(alias.Alias)
		}
	}
	for _, alias := range aliases {
		if !globalJSON {
			// Format properly for alignment based on alias length only in non json mode.
			alias.Alias = fmt.Sprintf("%-*.*s", maxAlias, maxAlias, alias.Alias)
		}
		if alias.AccessKey == "" || alias.SecretKey == "" {
			alias.AccessKey = ""
			alias.SecretKey = ""
			alias.API = ""
		}
		printMsg(alias)
	}
}

// byAlias is a collection satisfying sort.Interface
type byAlias []aliasMessage

func (d byAlias) Len() int           { return len(d) }
func (d byAlias) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d byAlias) Less(i, j int) bool { return d[i].Alias < d[j].Alias }

func buildAliasMessage(alias string, deprecated bool, aliasCfg *aliasConfigV10) aliasMessage {
	aliasMsg := aliasMessage{
		prettyPrint: false,
		Alias:       alias,
		URL:         aliasCfg.URL,
		AccessKey:   aliasCfg.AccessKey,
		SecretKey:   aliasCfg.SecretKey,
		API:         aliasCfg.API,
		Src:         aliasCfg.Src,
	}

	if deprecated {
		aliasMsg.Lookup = aliasCfg.Path
	} else {
		aliasMsg.Path = aliasCfg.Path
	}
	return aliasMsg
}

// listAliases - list one or all aliases
func listAliases(alias string, deprecated bool) (aliases []aliasMessage) {
	// If specific alias is requested, look for it and print.
	if alias != "" {
		aliasCfg := mustGetHostConfig(alias)
		if aliasCfg != nil {
			return []aliasMessage{buildAliasMessage(alias, deprecated, aliasCfg)}
		}

		fatalIf(errInvalidAliasedURL(alias), "No such alias `"+alias+"` found.")
	}

	// list alias from the environment variable.
	for _, envK := range env.List(mcEnvHostPrefix) {
		aliasCfg, _ := expandAliasFromEnv(env.Get(envK, ""))
		if aliasCfg == nil {
			continue
		}
		alias := strings.ReplaceAll(envK, mcEnvHostPrefix, "")
		aliases = append(aliases, buildAliasMessage(alias, deprecated, aliasCfg))
	}

	// list alias from the customized configuration.
	for s, aliasCfg := range aliasToConfigMap {
		aliases = append(aliases, buildAliasMessage(s, deprecated, aliasCfg))
	}

	// list alias from the default configuration.
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version `"+globalMCConfigVersion+"`.")

	for k, v := range conf.Aliases {
		v.Src = mustGetMcConfigPath()
		aliases = append(aliases, buildAliasMessage(k, deprecated, &v))
	}

	// Sort by alias names lexically.
	sort.Sort(byAlias(aliases))
	return
}
