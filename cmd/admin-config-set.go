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
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigSetCmd = cli.Command{
	Name:   "set",
	Usage:  "Set new config file to a Minio server/cluster.",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigSet,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set server configuration of a Minio server/cluster.
     $ cat myconfig | {{.HelpName}} myminio/

`,
}

// configSetMessage container to hold locks information.
type configSetMessage struct {
	Status          string `json:"status"`
	setConfigStatus bool
}

// String colorized service status message.
func (u configSetMessage) String() (msg string) {
	// Print the general set config status
	if u.setConfigStatus {
		msg += console.Colorize("SetConfigSuccess",
			"Setting new Minio configuration file has been successful.\n")
	} else {
		msg += console.Colorize("SetConfigFailure",
			"Setting new Minio configuration file has failed.\n")
	}
	return
}

// JSON jsonified service status Message message.
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
func checkAdminConfigSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
}

// main config set function
func mainAdminConfigSet(ctx *cli.Context) error {

	// Check command arguments
	checkAdminConfigSetSyntax(ctx)

	// Set color preference of command outputs
	console.SetColor("SetConfigSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("SetConfigFailure", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Call get config API
	fatalIf(probe.NewError(client.SetConfig(os.Stdin)), "Cannot set server configuration file.")

	// Print set config result
	printMsg(configSetMessage{
		setConfigStatus: true,
	})

	return nil
}
