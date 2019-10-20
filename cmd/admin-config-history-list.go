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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminConfigHistoryListCmd = cli.Command{
	Name:   "list",
	Usage:  "list all previously set keys on MinIO server",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigHistoryList,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all history entries sorted by set time.
     $ {{.HelpName}} play/
`,
}

// configHistoryListMessage container to hold locks information.
type configHistoryListMessage struct {
	Status  string                      `json:"status"`
	Entries []madmin.ConfigHistoryEntry `json:"entries"`
}

// String colorized service status message.
func (u configHistoryListMessage) String() string {
	var s []string
	for _, g := range u.Entries {
		message := console.Colorize("ConfigHistoryListMessageTime", fmt.Sprintf("[%s] ",
			g.CreateTime.Format(printDate)))
		message = message + console.Colorize("ConfigHistoryListMessageRestoreID", g.RestoreID)
		s = append(s, message)
	}
	return strings.Join(s, "\n")

}

// JSON jsonified service status Message message.
func (u configHistoryListMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigHistoryListSyntax - validate all the passed arguments
func checkAdminConfigHistoryListSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

func mainAdminConfigHistoryList(ctx *cli.Context) error {

	checkAdminConfigHistoryListSyntax(ctx)

	console.SetColor("ConfigHistoryListMessageRestoreID", color.New(color.Bold))
	console.SetColor("ConfigHistoryListMessageTime", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	chEntries, e := client.ListConfigHistoryKV()
	fatalIf(probe.NewError(e), "Cannot list server history configuration.")

	// Print
	printMsg(configHistoryListMessage{
		Entries: chEntries,
	})

	return nil
}
