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
	"encoding/base64"
	"io/ioutil"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var helpFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "env",
		Usage: "list all the env only help",
	},
}

var adminConfigHelpCmd = cli.Command{
	Name:   "help",
	Usage:  "show help for each sub-system and keys",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigHelp,
	Flags:  append(append([]cli.Flag{}, globalFlags...), helpFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Return help for 'region' settings on MinIO server.
     {{.Prompt}} {{.HelpName}} play/ region

  2. Return help for 'compression' settings, specifically 'extensions' key on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ compression extensions
`,
}

// configHelpMessage container to hold locks information.
type configHelpMessage struct {
	Status string `json:"status"`
	Value  string `json:"value"`
}

// String colorized service status message.
func (u configHelpMessage) String() string {
	return u.Value
}

// JSON jsonified service status Message message.
func (u configHelpMessage) JSON() string {
	u.Status = "success"
	u.Value = base64.StdEncoding.EncodeToString([]byte(u.Value))
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigHelpSyntax - validate all the passed arguments
func checkAdminConfigHelpSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) > 3 {
		cli.ShowCommandHelpAndExit(ctx, "help", 1) // last argument is exit code
	}
}

func mainAdminConfigHelp(ctx *cli.Context) error {

	checkAdminConfigHelpSyntax(ctx)

	// Help the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call get config API
	hr, e := client.HelpConfigKV(args.Get(1), args.Get(2), ctx.IsSet("env"))
	fatalIf(probe.NewError(e), "Cannot get help for the sub-system")

	buf, e := ioutil.ReadAll(hr)
	fatalIf(probe.NewError(e), "Cannot get help for the sub-system")

	// Print
	printMsg(configHelpMessage{
		Value: string(buf),
	})

	return nil
}
