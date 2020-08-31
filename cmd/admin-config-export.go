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
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigExportCmd = cli.Command{
	Name:   "export",
	Usage:  "export all config keys to STDOUT",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigExport,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Export the current config from MinIO server
     {{.Prompt}} {{.HelpName}} play/ > config.txt
`,
}

// configExportMessage container to hold locks information.
type configExportMessage struct {
	Status string `json:"status"`
	Value  []byte `json:"value"`
}

// String colorized service status message.
func (u configExportMessage) String() string {
	return string(u.Value)
}

// JSON jsonified service status Message message.
func (u configExportMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigExportSyntax - validate all the passed arguments
func checkAdminConfigExportSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "export", 1) // last argument is exit code
	}
}

func mainAdminConfigExport(ctx *cli.Context) error {

	checkAdminConfigExportSyntax(ctx)

	// Export the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call get config API
	buf, e := client.GetConfig(globalContext)
	fatalIf(probe.NewError(e), "Unable to get server config")

	// Print
	printMsg(configExportMessage{
		Value: buf,
	})

	return nil
}
