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

import (
	"errors"
	"fmt"
	"strings"

	"github.com/minio/mc/internal/github.com/minio/cli"
	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/internal/github.com/minio/minio/pkg/quick"
	"github.com/minio/mc/pkg/console"
)

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
var configCmd = cli.Command{
	Name:   "config",
	Usage:  "Modify, add alias, oauth into default configuration file [~/.mc/config.json]",
	Action: mainConfig,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} alias NAME HOSTURL

EXAMPLES:
   1. Add alias URLs.
      $ mc config alias zek https://s3.amazonaws.com/

`,
}

// mainConfig is the handle for "mc config" sub-command. writes configuration data in json format to config file.
func mainConfig(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	arg := ctx.Args().First()
	tailArgs := ctx.Args().Tail()
	if len(tailArgs) > 2 {
		fatalIf(probe.NewError(errors.New("")),
			"Incorrect number of arguments, please read ‘mc config help’")
	}
	configPath, err := getMcConfigPath()
	fatalIf(err, "Unable to get mc config path")

	switch arg {
	case "alias":
		if len(tailArgs) < 2 {
			cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
		}
		addAlias(tailArgs)
	default:
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	// upon success
	console.Infoln("Alias written successfully to [" + configPath + "].")
}

// addAlias - add new aliases
func addAlias(aliases []string) {
	conf := newConfigV1()
	config, err := quick.New(conf)
	fatalIf(err.Trace(), "Unable to initialize quick config")

	err = config.Load(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to load config path")

	aliasName := aliases[0]
	url := strings.TrimSuffix(aliases[1], "/")
	if strings.HasPrefix(aliasName, "http") {
		fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: Area51, Grand-Nagus..", aliasName))
	}
	if !strings.HasPrefix(url, "http") {
		fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Alias URL ‘%s’ is invalid, valid examples are: http://s3.amazonaws.com, https://yourbucket.example.com...", url))
	}
	if isAliasReserved(aliasName) {
		fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Alias name ‘%s’ is a reserved word, reserved words are [help, private, readonly, public, authenticated]", aliasName))
	}
	if !isValidAliasName(aliasName) {
		fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: Area51, Grand-Nagus..", aliasName))
	}
	// convert interface{} back to its original struct
	newConf := config.Data().(*configV1)
	if _, ok := newConf.Aliases[aliasName]; ok {
		fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Specified alias name: ‘%s’ already exists.", aliasName))
	}
	newConf.Aliases[aliasName] = url
	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(), "Unable to initialize quick config")

	err = writeConfig(newConfig)
	fatalIf(err.Trace(), "Unable to write alias name")
}
