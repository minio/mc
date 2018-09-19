/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var adminConfigGetCmd = cli.Command{
	Name:   "get",
	Usage:  "Get config of a Minio server/cluster.",
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
  1. Get server configuration of a Minio server/cluster.
     $ {{.HelpName}} play/

`,
}

// configGetMessage container to hold locks information.
type configGetMessage struct {
	Status string `json:"status"`
	Config string `json:"config"`
}

// String colorized service status message.
func (u configGetMessage) String() string {
	return string(u.Config)
}

// JSON jsonified service status Message message.
func (u configGetMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", "\t")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	// Remove \n and \t from u.Config which holds the config data
	return strings.NewReplacer(`\n`, "", `\t`, "").Replace(string(statusJSONBytes))
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

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Call get config API
	c, e := client.GetConfig()
	fatalIf(probe.NewError(e), "Cannot get server configuration file.")

	// Print
	printMsg(configGetMessage{Config: string(c)})

	return nil
}
