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
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminConfigEnvFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "env",
		Usage: "list all the env only help",
	},
}

var adminConfigResetCmd = cli.Command{
	Name:         "reset",
	Usage:        "interactively reset a config key parameters",
	Before:       setGlobalsFromContext,
	Action:       mainAdminConfigReset,
	OnUsageError: onUsageError,
	Flags:        append(adminConfigEnvFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Reset MQTT notifcation target 'name1' settings to default values.
     {{.Prompt}} {{.HelpName}} myminio/ notify_mqtt:name1
`,
}

// configResetMessage container to hold locks information.
type configResetMessage struct {
	Status      string `json:"status"`
	targetAlias string
}

// String colorized service status message.
func (u configResetMessage) String() (msg string) {
	msg += console.Colorize("ResetConfigSuccess",
		"Key is successfully reset.\n")
	suggestion := fmt.Sprintf("mc admin service restart %s", u.targetAlias)
	msg += console.Colorize("ResetConfigSuccess",
		fmt.Sprintf("Please restart your server with `%s`.\n", suggestion))
	return
}

// JSON jsonified service status message.
func (u configResetMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigResetSyntax - validate all the passed arguments
func checkAdminConfigResetSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "reset", 1) // last argument is exit code
	}
}

// main config set function
func mainAdminConfigReset(ctx *cli.Context) error {

	// Check command arguments
	checkAdminConfigResetSyntax(ctx)

	// Reset color preference of command outputs
	console.SetColor("ResetConfigSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("ResetConfigFailure", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if len(ctx.Args()) == 1 {
		// Call get config API
		hr, e := client.HelpConfigKV(globalContext, "", "", ctx.IsSet("env"))
		fatalIf(probe.NewError(e), "Unable to get help for the sub-system")

		// Print
		printMsg(configHelpMessage{
			Value:   hr,
			envOnly: ctx.IsSet("env"),
		})

		return nil
	}

	// Call reset config API
	input := strings.Join(args.Tail(), " ")
	fatalIf(probe.NewError(client.DelConfigKV(globalContext, input)),
		"Unable to reset '%s' on the server", input)

	// Print set config result
	printMsg(configResetMessage{
		targetAlias: aliasedURL,
	})

	return nil
}
