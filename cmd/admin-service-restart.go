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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var serviceRestartFlag = []cli.Flag{
	cli.BoolFlag{
		Name:  "dry-run",
		Usage: "do not attempt a restart, however verify the peer status",
	},
}

var adminServiceRestartCmd = cli.Command{
	Name:         "restart",
	Usage:        "restart a MinIO cluster",
	Action:       mainAdminServiceRestart,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(serviceRestartFlag, globalFlags...),
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
	Status    string                     `json:"status"`
	ServerURL string                     `json:"serverURL"`
	Result    madmin.ServiceActionResult `json:"result"`
}

// String colorized service restart command message.
func (s serviceRestartCommand) String() string {
	var s1 strings.Builder
	s1.WriteString("Restart command successfully sent to `" + s.ServerURL + "`. Type Ctrl-C to quit or wait to follow the status of the restart process.")

	if len(s.Result.Results) > 0 {
		s1.WriteString("\n")
		var rows []table.Row
		for _, peerRes := range s.Result.Results {
			errStr := tickCell
			if peerRes.Err != "" {
				errStr = peerRes.Err
			} else if len(peerRes.WaitingDrives) > 0 {
				errStr = fmt.Sprintf("%d drives are waiting for I/O and are offline, manual restart of OS is recommended", len(peerRes.WaitingDrives))
			}
			rows = append(rows, table.Row{peerRes.Host, errStr})
		}

		t := table.NewWriter()
		t.SetOutputMirror(&s1)
		t.SetColumnConfigs([]table.ColumnConfig{{Align: text.AlignCenter}})

		t.AppendHeader(table.Row{"Host", "Status"})
		t.AppendRows(rows)
		t.SetStyle(table.StyleLight)
		t.Render()
	}

	return console.Colorize("ServiceRestart", s1.String())
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
	result, e := client.ServiceAction(ctxt, madmin.ServiceActionOpts{
		Action: madmin.ServiceActionRestart,
		DryRun: ctx.Bool("dry-run"),
	})
	if e != nil {
		// Attempt an older API server might be old
		e = client.ServiceRestart(ctxt)
	}
	fatalIf(probe.NewError(e), "Unable to restart the server.")

	// Success..
	printMsg(serviceRestartCommand{Status: "success", ServerURL: aliasedURL, Result: result})

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

	t := time.Now()
	for {
		healthCtx, healthCancel := context.WithTimeout(ctxt, 2*time.Second)

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

		select {
		case <-ctxt.Done():
			return ctxt.Err()
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}
