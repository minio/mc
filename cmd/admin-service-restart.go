/*
 * MinIO Client (C) 2016 MinIO, Inc.
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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminServiceRestartCmd = cli.Command{
	Name:   "restart",
	Usage:  "restart all MinIO servers",
	Action: mainAdminServiceRestart,
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
	return console.Colorize("ServiceRestart", "Restart command successfully sent to `"+s.ServerURL+"`.")
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
		return console.Colorize("ServiceRestart", "Restarted `"+s.ServerURL+"` successfully.")
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

	// Max. time taken by the server to shutdown is 5 seconds.
	// This can happen when there are lot of s3 requests pending when the server
	// receives a restart command.
	// Sleep for 6 seconds and then check if the server is online.
	time.Sleep(6 * time.Second)

	// Fetch the service status of the specified MinIO server
	_, e := client.ServerInfo(globalContext)

	if e != nil {
		printMsg(serviceRestartMessage{Status: "failure", Err: e, ServerURL: aliasedURL})
	} else {
		printMsg(serviceRestartMessage{Status: "success", ServerURL: aliasedURL})
	}

	return nil
}
