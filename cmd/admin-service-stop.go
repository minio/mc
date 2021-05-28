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
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminServiceStopCmd = cli.Command{
	Name:         "stop",
	Usage:        "stop MinIO server",
	Action:       mainAdminServiceStop,
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
  1. Stop MinIO server represented by its alias 'play'.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// serviceStopMessage is container for make bucket success and failure messages.
type serviceStopMessage struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverURL"`
}

// String colorized make bucket message.
func (s serviceStopMessage) String() string {
	return console.Colorize("ServiceStop", "Stopped `"+s.ServerURL+"` successfully.")
}

// JSON jsonified make bucket message.
func (s serviceStopMessage) JSON() string {
	serviceStopJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serviceStopJSONBytes)
}

// checkAdminServiceStopSyntax - validate all the passed arguments
func checkAdminServiceStopSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "stop", 1) // last argument is exit code
	}
}

func mainAdminServiceStop(ctx *cli.Context) error {

	// Validate serivce stop syntax.
	checkAdminServiceStopSyntax(ctx)

	// Set color.
	console.SetColor("ServiceStop", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Stop the specified MinIO server
	fatalIf(probe.NewError(client.ServiceStop(globalContext)), "Unable to stop the server.")

	// Success..
	printMsg(serviceStopMessage{Status: "success", ServerURL: aliasedURL})
	return nil
}
