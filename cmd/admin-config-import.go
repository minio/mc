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
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminConfigImportCmd = cli.Command{
	Name:         "import",
	Usage:        "import multiple config keys from STDIN",
	Before:       setGlobalsFromContext,
	Action:       mainAdminConfigImport,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Import the new local config and apply to the MinIO server
     {{.Prompt}} {{.HelpName}} play/ < config.txt
`,
}

// configImportMessage container to hold locks information.
type configImportMessage struct {
	Status      string `json:"status"`
	targetAlias string
}

// String colorized service status message.
func (u configImportMessage) String() (msg string) {
	msg += console.Colorize("SetConfigSuccess",
		"Setting new key has been successful.\n")
	suggestion := fmt.Sprintf("mc admin service restart %s", u.targetAlias)
	msg += console.Colorize("SetConfigSuccess",
		fmt.Sprintf("Please restart your server with `%s`.\n", suggestion))
	return msg
}

// JSON jsonified service status Message message.
func (u configImportMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigImportSyntax - validate all the passed arguments
func checkAdminConfigImportSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "import", 1) // last argument is exit code
	}
}

func mainAdminConfigImport(ctx *cli.Context) error {

	checkAdminConfigImportSyntax(ctx)

	// Set color preference of command outputs
	console.SetColor("SetConfigSuccess", color.New(color.FgGreen, color.Bold))

	// Import the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call set config API
	fatalIf(probe.NewError(client.SetConfig(globalContext, os.Stdin)), "Unable to set server config")

	// Print
	printMsg(configImportMessage{
		targetAlias: aliasedURL,
	})

	return nil
}
