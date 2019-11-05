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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var historyListFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, n",
		Usage: "list only 'n' lines of history output",
		Value: 10,
	},
}

var adminConfigHistoryListCmd = cli.Command{
	Name:   "list",
	Usage:  "list all previously set keys on MinIO server",
	Before: setGlobalsFromContext,
	Action: mainAdminConfigHistoryList,
	Flags:  append(append([]cli.Flag{}, globalFlags...), historyListFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all history entries sorted by set time.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// HistoryList template used by all sub-systems
const HistoryList = `{{range .}}{{colorYellowBold "RestoreId:"}} {{colorYellowBold .RestoreID}}
Date: {{.CreateTime}}

{{.Targets}}

{{end}}`

// HistoryListTemplate - captures history list template
var HistoryListTemplate = template.Must(template.New("history-list").Funcs(funcMap).Parse(HistoryList))

type historyEntry struct {
	RestoreID  string         `json:"restoreId"`
	CreateTime string         `json:"createTime"`
	Targets    madmin.Targets `json:"targets"`
}

// configHistoryListMessage container to hold locks information.
type configHistoryListMessage struct {
	Status  string         `json:"status"`
	Entries []historyEntry `json:"entries"`
}

// String colorized service status message.
func (u configHistoryListMessage) String() string {
	var s strings.Builder
	w := tabwriter.NewWriter(&s, 1, 8, 2, ' ', 0)
	e := HistoryListTemplate.Execute(w, u.Entries)
	fatalIf(probe.NewError(e), "Cannot initialize template writer")

	w.Flush()
	return s.String()
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

	chEntries, e := client.ListConfigHistoryKV(ctx.Int("count"))
	fatalIf(probe.NewError(e), "Cannot list server history configuration.")

	hentries := make([]historyEntry, len(chEntries))
	for i, chEntry := range chEntries {
		hentries[i] = historyEntry{
			RestoreID:  chEntry.RestoreID,
			CreateTime: chEntry.CreateTimeFormatted(),
		}
		hentries[i].Targets, e = madmin.ParseSubSysTarget([]byte(chEntry.Data))
		fatalIf(probe.NewError(e), "Unable to parse invalid history entry.")
	}

	// Print
	printMsg(configHistoryListMessage{
		Entries: hentries,
	})

	return nil
}
