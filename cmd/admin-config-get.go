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
	"github.com/tidwall/gjson"
)

var adminConfigGetCmd = cli.Command{
	Name:   "get",
	Usage:  "Get configuration of a Minio server/cluster.",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigGet,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [key[.key] ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server configuration of a Minio server/cluster.
     $ {{.HelpName}} play/
  2. Get specific server configuration parameter(s) of a Minio server/cluster.
     $ {{.HelpName}} play/ region logger.console.enabled

`,
}

// configGetMessage container to hold status
// and Minio server config information
type configGetMessage struct {
	Status   string `json:"status"`
	Config   string `json:"config"`
	argsList []string
}

// String returns config info as a string
func (u configGetMessage) String() string {
	if len(u.argsList) == 0 {
		return u.Config
	}
	var str string
	for _, key := range u.argsList {
		val := gjson.Get(u.Config, key)
		str += key + " = " + val.Raw + "\n"
	}
	return str

}

// JSON jsonifies configuration GET message.
func (u configGetMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	// Remove \n and \t from u.Config which holds the config data
	return strings.NewReplacer(`\n`, "", `\t`, "").Replace(string(statusJSONBytes))
}

// checkAdminConfigGetSyntax - validates arguments
func checkAdminConfigGetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		cli.ShowCommandHelpAndExit(ctx, "get", 1) // last argument is exit code
	}
}

func mainAdminConfigGet(ctx *cli.Context) error {
	// Check command arguments
	checkAdminConfigGetSyntax(ctx)

	// Create a new Minio Admin Client
	// First argument, ctx.Args().Get(0), is the Minio server alias
	client, err := newAdminClient(ctx.Args().Get(0))
	fatalIf(err, "Cannot get a configured admin connection.")
	// Check number of arguments. If 1, then the whole file or
	// all configuration information will be the target. Otherwise,
	// individual configuration parameter(s) will be manipulated.
	if len(ctx.Args()) == 1 {
		// Call get config API
		c, e := client.GetConfig()
		fatalIf(probe.NewError(e), "Failed to get full configuration information.")
		printMsg(configGetMessage{Config: string(c)})
	} else {
		argsList := ctx.Args().Tail()
		// Call get config keys API
		c, e := client.GetConfigKeys(argsList)
		fatalIf(probe.NewError(e), "Failed to get configuration parameters: "+strings.Join(argsList, ", ")+".")
		printMsg(configGetMessage{Config: string(c),
			argsList: argsList})
	}
	return nil
}
