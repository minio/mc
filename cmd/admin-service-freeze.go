// Copyright (c) 2022 MinIO, Inc.
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
	"github.com/minio/pkg/v3/console"
)

var adminServiceFreezeCmd = cli.Command{
	Name:         "freeze",
	Usage:        "freeze S3 API calls on MinIO cluster",
	Action:       mainAdminServiceFreeze,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	Hidden:       true, // this command is hidden on purpose, please do not enable it.
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Freeze all S3 API calls on MinIO server at 'myminio/'.
     {{.Prompt}} {{.HelpName}} myminio/
`,
}

// serviceFreezeCommand is container for service freeze command success and failure messages.
type serviceFreezeCommand struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverURL"`
}

// String colorized service freeze command message.
func (s serviceFreezeCommand) String() string {
	return console.Colorize("ServiceFreeze", "Freeze command successfully sent to `"+s.ServerURL+"`.")
}

// JSON jsonified service freeze command message.
func (s serviceFreezeCommand) JSON() string {
	serviceFreezeJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serviceFreezeJSONBytes)
}

// checkAdminServiceFreezeSyntax - validate all the passed arguments
func checkAdminServiceFreezeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainAdminServiceFreeze(ctx *cli.Context) error {
	// Validate serivce freeze syntax.
	checkAdminServiceFreezeSyntax(ctx)

	// Set color.
	console.SetColor("ServiceFreeze", color.New(color.FgGreen, color.Bold))
	console.SetColor("FailedServiceFreeze", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Freeze the specified MinIO server
	fatalIf(probe.NewError(client.ServiceFreezeV2(globalContext)), "Unable to freeze the server.")

	// Success..
	printMsg(serviceFreezeCommand{Status: "success", ServerURL: aliasedURL})

	return nil
}
