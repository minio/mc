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
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminConfigSetCmd = cli.Command{
	Name:         "set",
	Usage:        "interactively set a config key parameters",
	Before:       setGlobalsFromContext,
	Action:       mainAdminConfigSet,
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
  1. Enable webhook notification target for MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ notify_webhook endpoint="http://localhost:8080/minio/events"

  2. Change region name for the MinIO server to 'us-west-1'.
     {{.Prompt}} {{.HelpName}} myminio/ region name=us-west-1

  3. Change healing settings on a distributed MinIO server setup.
     {{.Prompt}} {{.HelpName}} mydist/ heal max_delay=300ms max_io=50
`,
}

// configSetMessage container to hold locks information.
type configSetMessage struct {
	Status      string `json:"status"`
	targetAlias string
	restart     bool
}

// String colorized service status message.
func (u configSetMessage) String() (msg string) {
	msg += console.Colorize("SetConfigSuccess",
		"Successfully applied new settings.")
	if u.restart {
		suggestion := color.RedString("mc admin service restart %s", u.targetAlias)
		msg += console.Colorize("SetConfigSuccess",
			fmt.Sprintf("\nPlease restart your server '%s'.", suggestion))
	}
	return
}

// JSON jsonified service status message.
func (u configSetMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigSetSyntax - validate all the passed arguments
func checkAdminConfigSetSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() && len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
}

// main config set function
func mainAdminConfigSet(ctx *cli.Context) error {

	// Check command arguments
	checkAdminConfigSetSyntax(ctx)

	// Set color preference of command outputs
	console.SetColor("SetConfigSuccess", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	input := strings.Join(args.Tail(), " ")

	if !strings.Contains(input, madmin.KvSeparator) {
		// Call get config API
		hr, e := client.HelpConfigKV(globalContext, args.Get(1), args.Get(2), ctx.IsSet("env"))
		fatalIf(probe.NewError(e), "Unable to get help for the sub-system")

		// Print
		printMsg(configHelpMessage{
			Value:   hr,
			envOnly: ctx.IsSet("env"),
		})

		return nil

	}

	// Call set config API
	restart, e := client.SetConfigKV(globalContext, input)
	fatalIf(probe.NewError(e), "Unable to set '%s' to server", input)

	// Print set config result
	printMsg(configSetMessage{
		targetAlias: aliasedURL,
		restart:     restart,
	})

	return nil
}
