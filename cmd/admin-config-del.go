/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigDelCmd = cli.Command{
	Name:   "del",
	Usage:  "delete a key from MinIO server/cluster.",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigDel,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove MQTT notifcation target 'target1' on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ notify_mqtt:target1
`,
}

// configDelMessage container to hold locks information.
type configDelMessage struct {
	Status      string `json:"status"`
	targetAlias string
}

// String colorized service status message.
func (u configDelMessage) String() (msg string) {
	msg += console.Colorize("DelConfigSuccess",
		"Deleting key has been successful.\n")
	suggestion := fmt.Sprintf("mc admin service restart %s", u.targetAlias)
	msg += console.Colorize("DelConfigSuccess",
		fmt.Sprintf("Please restart your server with `%s`.\n", suggestion))
	return
}

// JSON jsonified service status message.
func (u configDelMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigDelSyntax - validate all the passed arguments
func checkAdminConfigDelSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "del", 1) // last argument is exit code
	}
}

// main config set function
func mainAdminConfigDel(ctx *cli.Context) error {

	// Check command arguments
	checkAdminConfigDelSyntax(ctx)

	// Del color preference of command outputs
	console.SetColor("DelConfigSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("DelConfigFailure", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call del config API
	input := strings.Join(args.Tail(), " ")
	if len(input) == 0 {
		b, e := ioutil.ReadAll(os.Stdin)
		fatalIf(probe.NewError(e), "Cannot read from the os.Stdin")
		input = string(b)
	}
	fatalIf(probe.NewError(client.DelConfigKV(input)),
		"Cannot delete '%s' on the server", input)

	// Print set config result
	printMsg(configDelMessage{
		targetAlias: aliasedURL,
	})

	return nil
}
