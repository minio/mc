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
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminServiceStopCmd = cli.Command{
	Name:   "stop",
	Usage:  "stop MinIO server",
	Action: mainAdminServiceStop,
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
