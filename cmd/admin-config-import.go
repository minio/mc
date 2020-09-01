/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminConfigImportCmd = cli.Command{
	Name:   "import",
	Usage:  "import multiple config keys from STDIN",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigImport,
	Flags:  globalFlags,
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
