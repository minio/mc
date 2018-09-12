/*
 * Minio Client (C) 2018 Minio, Inc.
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
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminConfigSetKeysCmd = cli.Command{
	Name:   "setkeys",
	Usage:  "Set partial Minio server/cluster configuration parameters using key/value pairs.",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigSetKeys,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [key1=value1 key2=value2 ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set a single Minio server/cluster configuration entry.
     $ {{.HelpName}} myminio region=us-east-1

  2. Set two Minio server/cluster configuration entries.
     $ {{.HelpName}} myminio logger.console.enabled=true logger.http.minio_one.enabled=true

  3. Set multiple Minio server/cluster configuration entries.
     $ {{.HelpName}} myminio logger.http.minio_one='{
                "enabled": true,
                "endpoint": "https://one.minio.io/logger/<some-jwt>"
            }'
`,
}

// configSetKeysMessage container to hold locks information.
type configSetKeysMessage struct {
	Status                   string `json:"status"`
	setKeysConfigStatus      bool
	SetKeysConfigNodeSummary []madmin.NodeSummary `json:"nodesSummary"`
}

// String colorized service status message.
func (u configSetKeysMessage) String() (msg string) {
	// For each node, print if set config was successful or not
	for _, s := range u.SetKeysConfigNodeSummary {
		if s.ErrSet {
			// An error occurred when setting the new config
			msg += console.Colorize("SetKeysConfigFailure",
				fmt.Sprintf("Failed to apply the new configuration to `%s` (%s).\n", s.Name, s.ErrMsg))
		} else {
			msg += console.Colorize("SetKeysConfigSuccess",
				fmt.Sprintf("New configuration applied to `%s` successfully.\n", s.Name))
		}
	}
	// Print the general set config status
	if u.setKeysConfigStatus {
		msg += console.Colorize("SetKeysConfigSuccess",
			"Setting new Minio configuration file has been successful.\n")
	} else {
		msg += console.Colorize("SetKeysConfigFailure",
			"Setting new Minio configuration file has failed.\n")
	}
	return
}

// JSON jsonified service status Message message.
func (u configSetKeysMessage) JSON() string {
	if u.setKeysConfigStatus {
		u.Status = "success"
	} else {
		u.Status = "error"
	}
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigSetKeysSyntax - validate all the passed arguments
func checkAdminConfigSetKeysSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		cli.ShowCommandHelpAndExit(ctx, "setkeys", 1) // last argument is exit code
	}
}

// main config setkeys function
func mainAdminConfigSetKeys(ctx *cli.Context) error {

	// Check command arguments
	checkAdminConfigSetKeysSyntax(ctx)

	// Set color preference of command outputs
	console.SetColor("SetKeysConfigSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("SetKeysConfigFailure", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	allArgs := ctx.Args()
	if len(allArgs) < 2 {
		fatalIf(probe.NewError(errors.New(
			"Usage: mc admin config setkeys alias key1=value1 [key2=value2 key3.key4=value4 ...]")), "Invalid number of arguments\n")
	}

	aliasedURL := allArgs.Get(0)
	argsMap := make(map[string]string)
	for _, arg := range allArgs.Tail() {
		argSplit := strings.SplitN(arg, "=", 2)
		if strings.Index(arg, "{") == -1 && len(argSplit)%2 == 1 {
			fatalIf(probe.NewError(errors.New(
				"Usage: mc admin config setkeys alias key1=value1 [key2=value2 key3.key4=value4 ...]")), "Invalid number of arguments\n")
		}
		argsMap[argSplit[0]] = argSplit[1]
	}

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize Admin client")

	// Call set config API
	// c, e := client.SetConfigKeys(argsMap)
	e := client.SetConfigKeys(argsMap)
	fatalIf(probe.NewError(e), "Cannot set server configuration.")

	// Print set config result
	// printMsg(configSetKeysMessage{
	// 	setKeysConfigStatus:      c.Status,
	// 	SetKeysConfigNodeSummary: c.NodeResults,
	// })

	return nil
}
