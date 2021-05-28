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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminServerUpdateCmd = cli.Command{
	Name:         "update",
	Usage:        "update all MinIO servers",
	Action:       mainAdminServerUpdate,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
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
