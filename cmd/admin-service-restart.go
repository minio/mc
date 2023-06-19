// Copyright (c) 2015-2022 MinIO, Inc.
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
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminServiceRestartCmd = cli.Command{
	Name:         "restart",
	Usage:        "restart a MinIO cluster",
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
	Status    string        `json:"status"`
	ServerURL string        `json:"serverURL"`
	TimeTaken time.Duration `json:"timeTaken"`
	Err       error         `json:"error,omitempty"`
}

// String colorized service restart message.
func (s serviceRestartMessage) String() string {
	if s.Err == nil {
		return console.Colorize("ServiceRestart", fmt.Sprintf("\nRestarted `%s` successfully in %s", s.ServerURL, timeDurationToHumanizedDuration(s.TimeTaken).StringShort()))
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
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainAdminServiceRestart(ctx *cli.Context) error {
	// Validate serivce restart syntax.
	checkAdminServiceRestartSyntax(ctx)

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

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
	fatalIf(probe.NewError(client.ServiceRestart(ctxt)), "Unable to restart the server.")

	// Success..
	printMsg(serviceRestartCommand{Status: "success", ServerURL: aliasedURL})

	// Start pinging the service until it is ready

	anonClient, err := newAnonymousClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Could not ping `"+aliasedURL+"`.")

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

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	t := time.Now()
	for {
		select {
		case <-ctxt.Done():
			return ctxt.Err()
		case <-timer.C:
			healthCtx, healthCancel := context.WithTimeout(ctxt, 3*time.Second)
			// Fetch the health status of the specified MinIO server
			healthResult, healthErr := anonClient.Healthy(healthCtx, madmin.HealthOpts{})
			healthCancel()
			switch {
			case healthErr == nil && healthResult.Healthy:
				printMsg(serviceRestartMessage{
					Status:    "success",
					ServerURL: aliasedURL,
					TimeTaken: time.Since(t),
				})
				return nil
			case healthErr == nil && !healthResult.Healthy:
				coloring = color.New(color.FgYellow)
				mark = "!"
				fallthrough
			default:
				printProgress()
			}

			timer.Reset(time.Second)
		}
	}
}
