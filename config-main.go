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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
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
	Usage:  "Modify, add, remove alias from default configuration file [~/.mc/config.json].",
	Action: mainConfig,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} OPERATION OPTION [ARGS...]

EXAMPLES:
   1. Add aliases for a URL
      $ mc {{.Name}} add alias mcloud https://s3.amazonaws.com/miniocloud
      $ mc ls mcloud
      $ mc cp /bin/true mcloud/true

   2. List all aliased URLs.
      $ mc {{.Name}} list alias

   3. Remove an alias
      $ mc {{.Name}} remove alias zek
`,
}

// AliasMessage container for content message structure
type AliasMessage struct {
	op    string
	Alias string `json:"alias"`
	URL   string `json:"url,omitempty"`
}

// String string printer for Content metadata
func (a AliasMessage) String() string {
	if !globalJSONFlag {
		if a.op == "list" {
			message := console.Colorize("Alias", fmt.Sprintf("[%s] <- ", a.Alias))
			message += console.Colorize("URL", fmt.Sprintf("%s", a.URL))
			return message
		}
		if a.op == "remove" {
			return console.Colorize("AliasMessage", "Removed alias ‘"+a.Alias+"’ successfully.")
		}
		if a.op == "add" {
			return console.Colorize("AliasMessage", "Added alias ‘"+a.Alias+"’ successfully.")
		}
	}
	jsonMessageBytes, e := json.Marshal(a)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func checkConfigSyntax(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	if len(ctx.Args().Tail()) > 3 {
		fatalIf(errDummy().Trace(), "Incorrect number of arguments to config command")
	}
	switch strings.TrimSpace(ctx.Args().First()) {
	case "add":
		if strings.TrimSpace(ctx.Args().Tail().First()) != "alias" {
			cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
		}
		if strings.TrimSpace(ctx.Args().Tail().First()) == "alias" {
			if len(ctx.Args().Tail().Tail()) != 2 {
				fatalIf(errInvalidArgument().Trace(), "Incorrect number of arguments for add alias command.")
			}
		}
	case "remove":
		if strings.TrimSpace(ctx.Args().Tail().First()) != "alias" {
			cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
		}
		if strings.TrimSpace(ctx.Args().Tail().First()) == "alias" {
			if len(ctx.Args().Tail().Tail()) != 1 {
				fatalIf(errInvalidArgument().Trace(), "Incorrect number of arguments for remove alias command.")
			}
		}
	case "list":
		if strings.TrimSpace(ctx.Args().Tail().First()) != "alias" {
			cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
}

// mainConfig is the handle for "mc config" sub-command. writes configuration data in json format to config file.
func mainConfig(ctx *cli.Context) {
	checkConfigSyntax(ctx)

	// set new custom coloring
	console.SetCustomTheme(map[string]*color.Color{
		"Alias":        color.New(color.FgCyan, color.Bold),
		"AliasMessage": color.New(color.FgGreen, color.Bold),
		"URL":          color.New(color.FgWhite),
	})

	arg := ctx.Args().First()
	tailArgs := ctx.Args().Tail()

	switch strings.TrimSpace(arg) {
	case "add":
		if strings.TrimSpace(tailArgs.First()) == "alias" {
			addAlias(tailArgs.Get(1), tailArgs.Get(2))
		}
	case "remove":
		if strings.TrimSpace(tailArgs.First()) == "alias" {
			removeAlias(tailArgs.Get(1))
		}
	case "list":
		if strings.TrimSpace(tailArgs.First()) == "alias" {
			listAliases()
		}
	}
}

// listAliases - list alias
func listAliases() {
	conf := newConfigV3()
	config, err := quick.New(conf)
	fatalIf(err.Trace(conf.Version), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV3)
	for k, v := range newConf.Aliases {
		console.Println(AliasMessage{
			op:    "list",
			Alias: k,
			URL:   v,
		})
	}
}

// removeAlias - remove alias
func removeAlias(alias string) {
	if alias == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	conf := newConfigV3()
	config, err := quick.New(conf)
	fatalIf(err.Trace(conf.Version), "Failed to initialize ‘quick’ configuration data structure.")

	err = config.Load(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to load config path")
	if !isValidAliasName(alias) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", alias))
	}
	// convert interface{} back to its original struct
	newConf := config.Data().(*configV3)

	if _, ok := newConf.Aliases[alias]; !ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias ‘%s’ does not exist.", alias))
	}
	delete(newConf.Aliases, alias)

	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(conf.Version), "Failed to initialize ‘quick’ configuration data structure.")

	err = writeConfig(newConfig)
	fatalIf(err.Trace(alias), "Unable to save alias ‘"+alias+"’.")

	console.Println(AliasMessage{
		op:    "remove",
		Alias: alias,
	})
}

// addAlias - add new aliases
func addAlias(alias, url string) {
	if alias == "" || url == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	conf := newConfigV3()
	config, err := quick.New(conf)
	fatalIf(err.Trace(conf.Version), "Failed to initialize ‘quick’ configuration data structure.")

	err = config.Load(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to load config path")

	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(url, "http") {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Invalid alias URL ‘%s’. Valid examples are: http://s3.amazonaws.com, https://yourbucket.example.com.", url))
	}
	if !isValidAliasName(alias) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", alias))
	}
	// convert interface{} back to its original struct
	newConf := config.Data().(*configV3)
	if oldURL, ok := newConf.Aliases[alias]; ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias ‘%s’ already exists for ‘%s’.", alias, oldURL))
	}
	newConf.Aliases[alias] = url
	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(conf.Version), "Failed to initialize ‘quick’ configuration data structure.")

	err = writeConfig(newConfig)
	fatalIf(err.Trace(alias, url), "Unable to save alias ‘"+alias+"’.")

	console.Println(AliasMessage{
		op:    "add",
		Alias: alias,
		URL:   url,
	})
}
