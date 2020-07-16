/*
 * MinIO Client (C) 2018-2019 MinIO, Inc.
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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminServerUpdateCmd = cli.Command{
	Name:   "update",
	Usage:  "update all MinIO servers",
	Action: mainAdminServerUpdate,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Update MinIO server represented by its alias 'play'.
     {{.Prompt}} {{.HelpName}} play/

  2. Update all MinIO servers in a distributed setup, represented by its alias 'mydist'.
     {{.Prompt}} {{.HelpName}} mydist/
`,
}

// serverUpdateMessage is container for ServerUpdate success and failure messages.
type serverUpdateMessage struct {
	Status         string `json:"status"`
	ServerURL      string `json:"serverURL"`
	CurrentVersion string `json:"currentVersion"`
	UpdatedVersion string `json:"updatedVersion"`
}

// String colorized serverUpdate message.
func (s serverUpdateMessage) String() string {
	if s.CurrentVersion != s.UpdatedVersion {
		return console.Colorize("ServerUpdate",
			fmt.Sprintf("Server `%s` updated successfully from %s to %s",
				s.ServerURL, s.CurrentVersion, s.UpdatedVersion))
	}
	return console.Colorize("ServerUpdate",
		fmt.Sprintf("Server `%s` already running the most recent version %s of MinIO",
			s.ServerURL, s.CurrentVersion))
}

// JSON jsonified server update message.
func (s serverUpdateMessage) JSON() string {
	serverUpdateJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serverUpdateJSONBytes)
}

// checkAdminServerUpdateSyntax - validate all the passed arguments
func checkAdminServerUpdateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}
}

func mainAdminServerUpdate(ctx *cli.Context) error {
	// Validate serivce update syntax.
	checkAdminServerUpdateSyntax(ctx)

	// Set color.
	console.SetColor("ServerUpdate", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	updateURL := args.Get(1)

	// Update the specified MinIO server, optionally also
	// with the provided update URL.
	us, e := client.ServerUpdate(globalContext, updateURL)
	fatalIf(probe.NewError(e), "Unable to update the server.")

	// Success..
	printMsg(serverUpdateMessage{
		Status:         "success",
		ServerURL:      aliasedURL,
		CurrentVersion: us.CurrentVersion,
		UpdatedVersion: us.UpdatedVersion,
	})
	return nil
}
