/*
 * MinIO Client (C) 2017 MinIO, Inc.
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

var adminConfigGetCmd = cli.Command{
	Name:   "get",
	Usage:  "get config of a MinIO server/cluster",
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
  1. Get server configuration of a MinIO server/cluster.
     $ {{.HelpName}} play/

`,
}

// configGetMessage container to hold locks information.
type configGetMessage struct {
	Status string                 `json:"status"`
	Config map[string]interface{} `json:"config"`
}

// String colorized service status message.
func (u configGetMessage) String() string {
	config, e := json.MarshalIndent(u.Config, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(config)
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
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
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
	fatalIf(err, "Cannot get a configured admin connection.")

	// Call get config API
	c, e := client.GetConfig()
	fatalIf(probe.NewError(e), "Cannot get server configuration file.")

	config := map[string]interface{}{}
	e = json.Unmarshal(c, &config)
	fatalIf(probe.NewError(e), "Cannot unmarshal server configuration file.")

	// Print
	printMsg(configGetMessage{
		Config: config,
	})

	return nil
}
