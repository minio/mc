/*
 * Minio Client (C) 2016, 2017 Minio, Inc.
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
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	adminServiceStatusFlags = []cli.Flag{}
)

var adminServiceStatusCmd = cli.Command{
	Name:   "status",
	Usage:  "Get the status of Minio server",
	Action: mainAdminServiceStatus,
	Before: setGlobalsFromContext,
	Flags:  append(adminServiceStatusFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Check if the 'play' Minio server is online and show its uptime.
       $ {{.HelpName}} play/
`,
}

// serviceStatusMessage container to hold service status information.
type serviceStatusMessage struct {
	Status  string        `json:"status"`
	Service string        `json:"service"`
	Uptime  time.Duration `json:"uptime"`
}

// String colorized service status message.
func (u serviceStatusMessage) String() (msg string) {
	defer func() {
		msg = console.Colorize("ServiceStatus", msg)
	}()
	// When service is offline
	if u.Service == "off" {
		msg = "The server is offline."
		return
	}
	msg = fmt.Sprintf("Uptime: %s.", timeDurationToHumanizedDuration(u.Uptime))
	return
}

// JSON jsonified service status Message message.
func (u serviceStatusMessage) JSON() string {
	switch u.Service {
	case "on":
		u.Status = "success"
	case "off":
		u.Status = "failure"
	}

	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminServiceStatusSyntax - validate all the passed arguments
func checkAdminServiceStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "status", 1) // last argument is exit code
	}
}

func mainAdminServiceStatus(ctx *cli.Context) error {

	// Validate serivce status syntax.
	checkAdminServiceStatusSyntax(ctx)

	console.SetColor("ServiceStatus", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Fetch the service status of the specified Minio server
	st, e := client.ServiceStatus()

	// Check the availability of the server: online or offline. A server is considered
	// offline if we can't get any response or we get a bad format response
	var serviceOffline bool
	switch v := e.(type) {
	case *json.SyntaxError:
		serviceOffline = true
	case *url.Error:
		if v.Timeout() {
			serviceOffline = true
		}
	}
	if serviceOffline {
		printMsg(serviceStatusMessage{Service: "off"})
		return nil
	}

	// If the error is not nil and not unrecognizable, just print it and exit
	fatalIf(probe.NewError(e), "Cannot get service status.")

	// Print the whole response
	printMsg(serviceStatusMessage{
		Service: "on",
		Uptime:  st.Uptime,
	})

	return nil
}
