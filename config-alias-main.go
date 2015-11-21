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
	Flags:  append(globalFlags, configAliasFlagHelp),
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

// checkConfigAliasAddSyntax validate 'config alias add' input arguments.
func checkConfigAliasAddSyntax(ctx *cli.Context) {
	if len(ctx.Args().Tail()) != 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for add alias command.")
	}
	aliasName := ctx.Args().Tail().Get(0)
	aliasedURL := ctx.Args().Tail().Get(1)
	if aliasName == "" || aliasedURL == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	aliasedURL = strings.TrimSuffix(aliasedURL, "/")
	if !strings.HasPrefix(aliasedURL, "http") {
		fatalIf(errDummy().Trace(aliasedURL),
			fmt.Sprintf("Invalid alias URL ‘%s’. Valid examples are: http://s3.amazonaws.com, https://yourbucket.example.com.", aliasedURL))
	}
	if !isValidAliasName(aliasName) {
		fatalIf(errDummy().Trace(aliasName),
			fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", aliasName))
	}
}

// checkConfigAliasRemoveSyntax validate 'config alias remove' input argument.
func checkConfigAliasRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args().Tail()) != 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for remove alias command.")
	}
	aliasName := ctx.Args().Tail().Get(0)
	if strings.TrimSpace(aliasName) == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	if !isValidAliasName(aliasName) {
		fatalIf(errDummy().Trace(aliasName),
			fmt.Sprintf("Alias name ‘%s’ is invalid, valid examples are: mybucket, Area51, Grand-Nagus", aliasName))
	}
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
		checkConfigAliasAddSyntax(ctx)
	case "remove":
		checkConfigAliasRemoveSyntax(ctx)
	case "list":
	default:
		cli.ShowCommandHelpAndExit(ctx, "alias", 1) // last argument is exit code
	}
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
		aliasName := tailArgs.Get(0)
		aliasedURL := tailArgs.Get(1)
		addAlias(aliasName, aliasedURL) // add alias name for aliased URL.
	case "remove":
		aliasName := tailArgs.Get(0)
		removeAlias(aliasName) // remove alias name.
	case "list":
		listAliases() // list all aliases.
	}
}

// addAlias - adds an alias entry.
func addAlias(aliasName, aliasedURL string) {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version ‘"+globalMCConfigVersion+"’.")

	conf.Aliases[aliasName] = aliasedURL

	err = saveMcConfig(conf)
	fatalIf(err.Trace(aliasName, aliasedURL), "Unable to update aliases in config version ‘"+globalMCConfigVersion+"’.")

	printMsg(aliasMessage{op: "add", Alias: aliasName, URL: aliasedURL})
}

// removeAlias - remove a alias.
func removeAlias(aliasName string) {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version ‘"+globalMCConfigVersion+"’.")

	delete(conf.Aliases, aliasName)

	err = saveMcConfig(conf)
	fatalIf(err.Trace(aliasName), "Unable to save deleted alias in config version ‘"+globalMCConfigVersion+"’.")

	printMsg(aliasMessage{op: "remove", Alias: aliasName})
}

// listAliases - list aliases.
func listAliases() {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version ‘"+globalMCConfigVersion+"’")

	for aliasName, aliasedURL := range conf.Aliases {
		printMsg(aliasMessage{op: "list", Alias: aliasName, URL: aliasedURL})
	}
}
