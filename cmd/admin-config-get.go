/*
 * MinIO Client (C) 2017-2019 MinIO, Inc.
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
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigGetCmd = cli.Command{
	Name:   "get",
	Usage:  "interactively retrieve a config key parameters",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigGet,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get the current region setting on MinIO server.
     {{.Prompt}} {{.HelpName}} play/ region
     region name=us-east-1

  2. Get the current notification settings for Webhook target on MinIO server
     {{.Prompt}} {{.HelpName}} myminio/ notify_webhook
     notify_webhook endpoint="http://localhost:8080" auth_token= queue_limit=10000 queue_dir="/home/events"

  3. Get the current compression settings on MinIO server
     {{.Prompt}} {{.HelpName}} myminio/ compression
     compression extensions=".txt,.csv" mime_types="text/*"
`,
}

// configGetMessage container to hold locks information.
type configGetMessage struct {
	Status string `json:"status"`
	Value  []byte `json:"value"`
}

// String colorized service status message.
func (u configGetMessage) String() string {
	return string(u.Value)
}

// JSON jsonified service status Message message.
func (u configGetMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigGetSyntax - validate all the passed arguments
func checkAdminConfigGetSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "get", 1) // last argument is exit code
	}
}

func mainAdminConfigGet(ctx *cli.Context) error {

	checkAdminConfigGetSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if len(ctx.Args()) == 1 {
		// Call get config API
		hr, e := client.HelpConfigKV(globalContext, "", "", false)
		fatalIf(probe.NewError(e), "Unable to get help for the sub-system")

		// Print
		printMsg(configHelpMessage{
			Value:   hr,
			envOnly: false,
		})

		return nil
	}

	// Call get config API
	buf, e := client.GetConfigKV(globalContext, strings.Join(args.Tail(), " "))
	fatalIf(probe.NewError(e), "Unable to get server '%s' config", args.Tail())

	// Print
	printMsg(configGetMessage{
		Value: buf,
	})

	return nil
}
