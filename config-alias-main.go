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

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

var (
	configAliasFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of config alias.",
	}
)

var configAliasCmd = cli.Command{
	Name:   "alias",
	Usage:  "List, modify and remove aliases in configuration file.",
	Action: mainConfigAlias,
	Flags:  []cli.Flag{configAliasFlagHelp},
	CustomHelpTemplate: `NAME:
   mc config {{.Name}} - {{.Usage}}

USAGE:
   mc config {{.Name}} [FLAGS] OPERATION [ARGS...]

OPERATION:
   remove   Remove a alias.
   list     List all aliases.
   add      Add new alias.

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Add aliases for Amazon S3.
      $ mc config {{.Name}} add mcloud https://miniocloud.s3.amazonaws.com
      $ mc ls mcloud

   2. Add aliases for Google Cloud Storage.
      $ mc config {{.Name}} add gcscloud https://miniocloud.storage.googleapis.com
      $ mc ls gcscloud

   3. List all aliased URLs.
      $ mc config {{.Name}} list

   4. Remove an alias
      $ mc config {{.Name}} remove mcloud

`,
}

// aliasMessage container for content message structure
type aliasMessage struct {
	op     string
	Status string `json:"status"`
	Alias  string `json:"alias"`
	URL    string `json:"url,omitempty"`
}

// String colorized alias message
func (a aliasMessage) String() string {
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
func (a aliasMessage) JSON() string {
	a.Status = "success"
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
		fatalIf(errDummy().Trace(ctx.Args().Tail()...), "Incorrect number of arguments to alias command")
	}
	switch strings.TrimSpace(ctx.Args().Get(0)) {
	case "add":
		if len(ctx.Args().Tail()) != 2 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...), "Incorrect number of arguments for add alias command.")
		}
	case "remove":
		if len(ctx.Args().Tail()) != 1 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...), "Incorrect number of arguments for remove alias command.")
		}
	case "list":
	default:
		cli.ShowCommandHelpAndExit(ctx, "alias", 1) // last argument is exit code
	}
}

func listAliases() {
	config, err := newConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV6)
	for k, v := range newConf.Aliases {
		printMsg(aliasMessage{op: "list", Alias: k, URL: v})
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
		fatalIf(errDummy().Trace(alias), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", alias))
	}

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV6)
	if _, ok := newConf.Aliases[alias]; !ok {
		fatalIf(errDummy().Trace(alias), fmt.Sprintf("Alias ‘%s’ does not exist.", alias))
	}
	delete(newConf.Aliases, alias)

	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")
	err = writeConfig(newConfig)
	fatalIf(err.Trace(alias), "Unable to save alias ‘"+alias+"’.")

	printMsg(aliasMessage{op: "remove", Alias: alias})
}

// addAlias - add new aliases
func addAlias(alias, url string) {
	if alias == "" || url == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	config, err := newConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	err = config.Load(mustGetMcConfigPath())
	fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to load config path")

	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(url, "http") {
		fatalIf(errDummy().Trace(url), fmt.Sprintf("Invalid alias URL ‘%s’. Valid examples are: http://s3.amazonaws.com, https://yourbucket.example.com.", url))
	}
	if !isValidAliasName(alias) {
		fatalIf(errDummy().Trace(alias), fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", alias))
	}
	// convert interface{} back to its original struct
	newConf := config.Data().(*configV6)
	if oldURL, ok := newConf.Aliases[alias]; ok {
		fatalIf(errDummy().Trace(alias), fmt.Sprintf("Alias ‘%s’ already exists for ‘%s’.", alias, oldURL))
	}
	newConf.Aliases[alias] = url
	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	err = writeConfig(newConfig)
	fatalIf(err.Trace(alias, url), "Unable to save alias ‘"+alias+"’.")

	printMsg(aliasMessage{op: "add", Alias: alias, URL: url})
}

func mainConfigAlias(ctx *cli.Context) {
	checkConfigAliasSyntax(ctx)

	// Additional customization speific to each command.
	console.SetColor("Alias", color.New(color.FgCyan, color.Bold))
	console.SetColor("AliasMessage", color.New(color.FgGreen, color.Bold))
	console.SetColor("URL", color.New(color.FgWhite, color.Bold))

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
