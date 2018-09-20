/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigSetCmd = cli.Command{
	Name:   "set",
	Usage:  "Set Minio server/cluster configuration.",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigSet,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [key[.key ...]=value]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set server configuration of a Minio server/cluster.
     $ cat myconfig | {{.HelpName}} myminio/
  2. Set a single Minio server/cluster configuration entry.
     $ {{.HelpName}} myminio region=us-east-1

  3. Set two Minio server/cluster configuration entries.
     $ {{.HelpName}} myminio logger.console.enabled=true cache.expiry=24

`,
}

// configSetMessage container to hold status
type configSetMessage struct {
	Status          string `json:"status"`
	setConfigStatus bool
}

// String colorizes config set messages
func (u configSetMessage) String() (msg string) {
	// Print the general set config status
	if u.setConfigStatus {
		msg += console.Colorize("SetConfigSuccess",
			"Successfully set Minio server configuration.\n")
	} else {
		msg += console.Colorize("SetConfigFailure",
			"Failed to set Minio server configuration.\n")
	}
	return
}

// JSON jsonifies config SET message.
func (u configSetMessage) JSON() string {
	if u.setConfigStatus {
		u.Status = "success"
	} else {
		u.Status = "error"
	}
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigSetSyntax - validate all the passed arguments
func checkAdminConfigSetSyntax(ctx *cli.Context) string {
	if len(ctx.Args()) == 0 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
	if len(ctx.Args()) == 1 {
		return "fullSet"
	}
	// if 2 or more arguments are passed
	return "partialSet"
}

// main config set function
func mainAdminConfigSet(ctx *cli.Context) error {
	// Initializations
	aliasedURL := ctx.Args().Get(0)
	isFullConfigSet := false

	// Check command arguments
	if checkAdminConfigSetSyntax(ctx) == "fullSet" {
		isFullConfigSet = true
	}

	// Set color preference of command outputs
	console.SetColor("SetConfigSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("SetConfigFailure", color.New(color.FgRed, color.Bold))

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")
	if isFullConfigSet {
		// Call set config API
		fatalIf(probe.NewError(client.SetConfig(os.Stdin)), "Cannot set server configuration file.")
		// Print set config result
		printMsg(configSetMessage{setConfigStatus: true})
	} else {
		argsMap := make(map[string]string)
		for _, arg := range ctx.Args().Tail() {
			argSplit := strings.SplitN(arg, "=", 2)
			if strings.Index(arg, "{") == -1 && len(argSplit)%2 == 1 {
				fatalIf(probe.NewError(errors.New(
					"Usage: mc admin config setkeys TARGET [key[.key ...]=value]")), "Invalid number of arguments\n")
			}
			argsMap[argSplit[0]] = argSplit[1]
		}
		// Call set config API
		fatalIf(probe.NewError(client.SetConfigKeys(argsMap)), "Cannot set server configuration parameter(s).")
		// Print set config result
		printMsg(configSetMessage{setConfigStatus: true})
	}

	return nil
}
