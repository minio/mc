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
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigGetCmd = cli.Command{
	Name:         "get",
	Usage:        "interactively retrieve a config key parameters",
	Before:       setGlobalsFromContext,
	Action:       mainAdminConfigGet,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
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
