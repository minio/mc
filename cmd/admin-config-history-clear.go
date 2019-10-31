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
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigHistoryClearCmd = cli.Command{
	Name:   "clear",
	Usage:  "clear a history key value on MinIO server",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigHistoryClear,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET RESTOREID

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Clear a history key value on MinIO server.
     {{.Prompt}} {{.HelpName}} play/ <restore-id>
`,
}

// configHistoryClearMessage container to hold locks information.
type configHistoryClearMessage struct {
	Status    string `json:"status"`
	RestoreID string `json:"restoreID"`
}

// String colorized service status message.
func (u configHistoryClearMessage) String() string {
	if u.RestoreID == "all" {
		return console.Colorize("ConfigHistoryClearMessage", "Cleared all keys successfully.")
	}
	return console.Colorize("ConfigHistoryClearMessage", "Cleared "+u.RestoreID+" successfully.")
}

// JSON jsonified service status Message message.
func (u configHistoryClearMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigHistoryClearSyntax - validate all the passed arguments
func checkAdminConfigHistoryClearSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "clear", 1) // last argument is exit code
	}
}

func mainAdminConfigHistoryClear(ctx *cli.Context) error {

	checkAdminConfigHistoryClearSyntax(ctx)

	console.SetColor("ConfigHistoryClearMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	fatalIf(probe.NewError(client.ClearConfigHistoryKV(args.Get(1))), "Cannot clear server configuration.")

	// Print
	printMsg(configHistoryClearMessage{
		RestoreID: args.Get(1),
	})

	return nil
}
