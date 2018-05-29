/*
 * Minio Client (C) 2016 Minio, Inc.
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

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminServiceRestartCmd = cli.Command{
	Name:   "restart",
	Usage:  "Restart Minio server",
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
    1. Restart Minio server represented by its alias 'play'.
       $ {{.HelpName}} play/

`,
}

// serviceRestartMessage is container for make bucket success and failure messages.
type serviceRestartMessage struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverURL"`
}

// String colorized make bucket message.
func (s serviceRestartMessage) String() string {
	return console.Colorize("ServiceRestart", "Restarted `"+s.ServerURL+"` successfully.")
}

// JSON jsonified make bucket message.
func (s serviceRestartMessage) JSON() string {
	serviceRestartJSONBytes, e := json.Marshal(s)
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

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Restart the specified Minio server
	fatalIf(probe.NewError(client.ServiceSendAction(
		madmin.ServiceActionValueRestart)), "Cannot restart server.")

	// Success..
	printMsg(serviceRestartMessage{Status: "success", ServerURL: aliasedURL})
	return nil
}
