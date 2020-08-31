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
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var historyListFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, n",
		Usage: "list only last 'n' entries",
		Value: 10,
	},
	cli.BoolFlag{
		Name:  "clear, c",
		Usage: "clear all history",
	},
}

var adminConfigHistoryCmd = cli.Command{
	Name:   "history",
	Usage:  "show all historic configuration changes",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigHistory,
	Flags:  append(append([]cli.Flag{}, globalFlags...), historyListFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all history entries sorted by time.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// History template used by all sub-systems
const History = `{{range .}}{{colorYellowBold "RestoreId:"}} {{colorYellowBold .RestoreID}}
Date: {{.CreateTime}}

{{.Targets}}

{{end}}`

// HistoryTemplate - captures history list template
var HistoryTemplate = template.Must(template.New("history-list").Funcs(funcMap).Parse(History))

type historyEntry struct {
	RestoreID  string `json:"restoreId"`
	CreateTime string `json:"createTime"`
	Targets    string `json:"targets"`
}

// configHistoryMessage container to hold locks information.
type configHistoryMessage struct {
	Status  string         `json:"status"`
	Entries []historyEntry `json:"entries"`
}

// String colorized service status message.
func (u configHistoryMessage) String() string {
	var s strings.Builder
	w := tabwriter.NewWriter(&s, 1, 8, 2, ' ', 0)
	e := HistoryTemplate.Execute(w, u.Entries)
	fatalIf(probe.NewError(e), "Unable to initialize template writer")

	w.Flush()
	return s.String()
}

// JSON jsonified service status Message message.
func (u configHistoryMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigHistorySyntax - validate all the passed arguments
func checkAdminConfigHistorySyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "history", 1) // last argument is exit code
	}
}

func mainAdminConfigHistory(ctx *cli.Context) error {

	checkAdminConfigHistorySyntax(ctx)

	console.SetColor("ConfigHistoryMessageRestoreID", color.New(color.Bold))
	console.SetColor("ConfigHistoryMessageTime", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if ctx.IsSet("clear") {
		fatalIf(probe.NewError(client.ClearConfigHistoryKV(globalContext, "all")), "Unable to clear server configuration.")

		// Print
		printMsg(configHistoryMessage{})
		return nil
	}

	chEntries, e := client.ListConfigHistoryKV(globalContext, ctx.Int("count"))
	fatalIf(probe.NewError(e), "Unable to list server history configuration.")

	hentries := make([]historyEntry, len(chEntries))
	for i, chEntry := range chEntries {
		hentries[i] = historyEntry{
			RestoreID:  chEntry.RestoreID,
			CreateTime: chEntry.CreateTimeFormatted(),
		}
		hentries[i].Targets = chEntry.Data
	}

	// Print
	printMsg(configHistoryMessage{
		Entries: hentries,
	})

	return nil
}
