/*
 * Minio Client (C) 2015 Minio, Inc.
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

// listAliases - list alias
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

var configAliasCmd = cli.Command{
	Name:   "alias",
	Usage:  "List, modify and remove aliases in configuration file.",
	Action: mainConfigAlias,
	CustomHelpTemplate: `NAME:
   mc config {{.Name}} - {{.Usage}}

USAGE:
   mc config {{.Name}} OPERATION [ARGS...]

   OPERATION = add | list | remove

EXAMPLES:
   1. Add aliases for a URL
      $ mc config {{.Name}} add mcloud https://s3.amazonaws.com/miniocloud
      $ mc ls mcloud
      $ mc cp /bin/true mcloud/true

   2. List all aliased URLs.
      $ mc config {{.Name}} list

   3. Remove an alias
      $ mc config {{.Name}} remove zek

`,
}

// AliasMessage container for content message structure
type AliasMessage struct {
	op    string
	Alias string `json:"alias"`
	URL   string `json:"url,omitempty"`
}

// String colorized alias message
func (a AliasMessage) String() string {
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
	// should never come here
	return ""
}

// JSON jsonified alias message
func (a AliasMessage) JSON() string {
	jsonMessageBytes, e := json.Marshal(a)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func checkConfigAliasSyntax(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "alias", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "alias", 1) // last argument is exit code
	}
	if len(ctx.Args().Tail()) > 2 {
		fatalIf(errDummy().Trace(), "Incorrect number of arguments to alias command")
	}
	switch strings.TrimSpace(ctx.Args().Get(0)) {
	case "add":
		if len(ctx.Args().Tail()) != 2 {
			fatalIf(errInvalidArgument().Trace(), "Incorrect number of arguments for add alias command.")
		}
	case "remove":
		if len(ctx.Args().Tail()) != 1 {
			fatalIf(errInvalidArgument().Trace(), "Incorrect number of arguments for remove alias command.")
		}
	case "list":
	default:
		cli.ShowCommandHelpAndExit(ctx, "alias", 1) // last argument is exit code
	}
}

func setConfigAliasPalette(style string) {
	console.SetCustomPalette(map[string]*color.Color{
		"Alias":        color.New(color.FgCyan, color.Bold),
		"AliasMessage": color.New(color.FgGreen, color.Bold),
		"URL":          color.New(color.FgWhite, color.Bold),
	})
	if style == "light" {
		console.SetCustomPalette(map[string]*color.Color{
			"Alias":        color.New(color.FgWhite, color.Bold),
			"AliasMessage": color.New(color.FgWhite, color.Bold),
			"URL":          color.New(color.FgWhite, color.Bold),
		})
		return
	}
	/// Add more styles here
	if style == "nocolor" {
		// All coloring options exhausted, setting nocolor safely
		console.SetNoColor()
	}
}

//
func mainConfigAlias(ctx *cli.Context) {
	checkConfigAliasSyntax(ctx)

	setConfigAliasPalette(ctx.GlobalString("colors"))

	arg := ctx.Args().First()
	tailArgs := ctx.Args().Tail()

	switch strings.TrimSpace(arg) {
	case "add":
		addAlias(tailArgs.Get(0), tailArgs.Get(1))
	case "remove":
		removeAlias(tailArgs.Get(0))
	case "list":
		listAliases()
	}
}

func listAliases() {
	config, err := newConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV5)
	for k, v := range newConf.Aliases {
		Prints("%s\n", AliasMessage{
			op:    "list",
			Alias: k,
			URL:   v,
		})
	}
}

// removeAlias - remove alias
func removeAlias(alias string) {
	if strings.TrimSpace(alias) == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	config, err := newConfig()
	fatalIf(err.Trace(), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")
	if !isValidAliasName(alias) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", alias))
	}

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV5)
	if _, ok := newConf.Aliases[alias]; !ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias ‘%s’ does not exist.", alias))
	}
	delete(newConf.Aliases, alias)

	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")
	err = writeConfig(newConfig)
	fatalIf(err.Trace(alias), "Unable to save alias ‘"+alias+"’.")

	Prints("%s\n", AliasMessage{
		op:    "remove",
		Alias: alias,
	})
}

// addAlias - add new aliases
func addAlias(alias, url string) {
	if alias == "" || url == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	config, err := newConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

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
	newConf := config.Data().(*configV5)
	if oldURL, ok := newConf.Aliases[alias]; ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Alias ‘%s’ already exists for ‘%s’.", alias, oldURL))
	}
	newConf.Aliases[alias] = url
	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	err = writeConfig(newConfig)
	fatalIf(err.Trace(alias, url), "Unable to save alias ‘"+alias+"’.")

	Prints("%s\n", AliasMessage{
		op:    "add",
		Alias: alias,
		URL:   url,
	})
}
