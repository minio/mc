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
	"context"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminServiceRestartCmd = cli.Command{
	Name:         "restart",
	Usage:        "restart all MinIO servers",
	Action:       mainAdminServiceRestart,
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
  1. Restart MinIO server represented by its alias 'play'.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// serviceRestartCommand is container for service restart command success and failure messages.
type serviceRestartCommand struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverURL"`
}

// String colorized service restart command message.
func (s serviceRestartCommand) String() string {
	return console.Colorize("ServiceRestart", "Restart command successfully sent to `"+s.ServerURL+"`. Type Ctrl-C to quit or wait to follow the status of the restart process.")
}

// JSON jsonified service restart command message.
func (s serviceRestartCommand) JSON() string {
	serviceRestartJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serviceRestartJSONBytes)
}

// serviceRestartMessage is container for service restart success and failure messages.
type serviceRestartMessage struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverURL"`
	Err       error  `json:"error,omitempty"`
}

// String colorized service restart message.
func (s serviceRestartMessage) String() string {
	if s.Err == nil {
		return console.Colorize("ServiceRestart", "\nRestarted `"+s.ServerURL+"` successfully.")
	}
	return console.Colorize("FailedServiceRestart", "Failed to restart `"+s.ServerURL+"`. error: "+s.Err.Error())
}

// JSON jsonified service restart message.
func (s serviceRestartMessage) JSON() string {
	serviceRestartJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serviceRestartJSONBytes)
}

// checkAdminServiceRestartSyntax - validate all the passed arguments
func checkAdminServiceRestartSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "restart", 1) // last argument is exit code
	}
}

func mainAdminServiceRestart(ctx *cli.Context) error {

	// Validate serivce restart syntax.
	checkAdminServiceRestartSyntax(ctx)

	// Set color.
	console.SetColor("ServiceOffline", color.New(color.FgRed, color.Bold))
	console.SetColor("ServiceInitializing", color.New(color.FgYellow, color.Bold))
	console.SetColor("ServiceRestart", color.New(color.FgGreen, color.Bold))
	console.SetColor("FailedServiceRestart", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Restart the specified MinIO server
	fatalIf(probe.NewError(client.ServiceRestart(globalContext)), "Unable to restart the server.")

	// Success..
	printMsg(serviceRestartCommand{Status: "success", ServerURL: aliasedURL})

	coloring := color.New(color.FgRed)
	mark := "..."

	// Print restart progress
	printProgress := func() {
		if !globalQuiet && !globalJSON {
			coloring.Printf(mark)
		}
	}

	printProgress()
	mark = "."

	for {
		select {
		case <-globalContext.Done():
			return globalContext.Err()
		case <-time.NewTimer(3 * time.Second).C:
			ctx, cancel := context.WithTimeout(globalContext, 1*time.Second)
			// Fetch the service status of the specified MinIO server
			info, e := client.ServerInfo(ctx)
			cancel()
			switch {
			case e == nil && info.Mode == string(madmin.ItemOnline):
				printMsg(serviceRestartMessage{Status: "success", ServerURL: aliasedURL})
				return nil
			case err == nil && info.Mode == string(madmin.ItemInitializing):
				coloring = color.New(color.FgYellow)
				mark = "!"
				fallthrough
			default:
				printProgress()
			}
		}
	}
}
