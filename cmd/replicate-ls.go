/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/gdamore/tcell"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
	"github.com/rivo/tview"
)

var replicateListFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "status",
		Usage: "show rules by status. Valid options are [enabled,disabled]",
	},
}

var replicateListCmd = cli.Command{
	Name:         "ls",
	Usage:        "list server side replication configuration rules",
	Action:       mainReplicateList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateListFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
	 
USAGE:
  {{.HelpName}} TARGET
	 
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List server side replication configuration rules on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkReplicateListSyntax - validate all the passed arguments
func checkReplicateListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
}

type replicateListMessage struct {
	Op     string           `json:"op"`
	Status string           `json:"status"`
	URL    string           `json:"url"`
	Rule   replication.Rule `json:"rule"`
}

func (l replicateListMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateListMessage) String() string {
	idFieldMaxLen := 20
	priorityFieldMaxLen := 8
	statusFieldMaxLen := 8
	prefixFieldMaxLen := 25
	tagsFieldMaxLen := 25
	scFieldMaxLen := 15
	destBucketFieldMaxLen := 20
	r := l.Rule
	return console.Colorize("replicateListMessage", newPrettyTable(" | ",
		Field{"ID", idFieldMaxLen},
		Field{"Priority", priorityFieldMaxLen},
		Field{"Status", statusFieldMaxLen},
		Field{"Prefix", prefixFieldMaxLen},
		Field{"Tags", tagsFieldMaxLen},
		Field{"DestBucket", destBucketFieldMaxLen},
		Field{"StorageClass", scFieldMaxLen},
	).buildRow(r.ID, strconv.Itoa(r.Priority), string(r.Status), r.Filter.And.Prefix, r.Tags(), r.Destination.Bucket, r.Destination.StorageClass))
}

func mainReplicateList(cliCtx *cli.Context) error {
	ctx, cancelReplicateList := context.WithCancel(globalContext)
	defer cancelReplicateList()

	console.SetColor("Headers", color.New(color.Bold, color.FgHiGreen))

	checkReplicateListSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	rCfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication configuration")

	if rCfg.Empty() {
		fatalIf(probe.NewError(errors.New("replication configuration not set")).Trace(aliasedURL),
			"Unable to list replication configuration")
	}
	printReplicateListing(rCfg, cliCtx.String("status"))
	return nil
}

func printReplicateListing(rCfg replication.Config, ruleStatus string) {
	if globalJSON {
		for _, rule := range rCfg.Rules {
			if ruleStatus == "" || strings.EqualFold(ruleStatus, string(rule.Status)) {
				printMsg(replicateListMessage{
					Rule: rule,
				})
			}
		}
		return
	}
	var rules []replication.Rule
	for _, r := range rCfg.Rules {
		if ruleStatus == "" || string(r.Status) == ruleStatus {
			rules = append(rules, r)
		}
	}
	if len(rules) == 0 {
		return
	}
	app := tview.NewApplication()
	table := tview.NewTable().
		SetBorders(true)
	table.SetBorderAttributes(tcell.AttrDim)

	header := strings.Split("ID,Priority,Status,Prefix,Tags,DestBucket,StorageClass", ",")
	cols, rows := len(header), len(rCfg.Rules)+1
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			table.SetCell(r, c, getCell(header, rules, r, c))
		}
	}
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		k := event.Key()
		if k == tcell.KeyCtrlC {
			app.Stop()
			os.Exit(0)
		}
		if k == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				app.Stop()
				os.Exit(0)
			}
		}
		return event
	})

	grid := tview.NewGrid().SetRows(3, 20).SetBorders(false)
	frame := tview.NewFrame(tview.NewBox()).
		SetBorders(1, 1, 1, 1, 1, 1).
		AddText("Replication Arn:", true, tview.AlignLeft, tcell.ColorGreenYellow).
		AddText(rCfg.Role, true, tview.AlignRight, tcell.ColorWhite).
		AddText("<ctrl-c> Quit", true, tview.AlignRight, tcell.ColorSlateGray)

	grid.AddItem(frame, 0, 0, 3, 1, 0, 100, false)
	grid.AddItem(table, 1, 0, 1, 1, 0, 0, true)

	if err := app.SetRoot(grid, true).EnableMouse(false).Run(); err != nil {
		panic(err)
	}
}
func getCell(header []string, rules []replication.Rule, r, c int) *tview.TableCell {
	var cell *tview.TableCell
	if r == 0 {
		cell = &tview.TableCell{
			Text:          header[c],
			Color:         tcell.ColorSilver,
			Align:         tview.AlignCenter,
			NotSelectable: true,
			Attributes:    tcell.AttrBold,
			Expansion:     1,
		}
		cell.SetTextColor(tcell.ColorGreenYellow)
		return cell
	}
	cell = &tview.TableCell{
		Color:         tcell.ColorGreenYellow,
		Align:         tview.AlignLeft,
		NotSelectable: true,
		Expansion:     1,
	}
	cell.SetTextColor(tcell.ColorSilver)

	maxWidth := 0
	switch c {
	case 0:
		maxWidth = 20
		cell.SetText(rules[r-1].ID)
	case 1:
		maxWidth = 8
		cell.SetText(strconv.Itoa(rules[r-1].Priority))
	case 2:
		maxWidth = 8
		cell.SetText(string(rules[r-1].Status))
	case 3:
		maxWidth = 25
		cell.SetText(rules[r-1].Filter.And.Prefix)
	case 4:
		maxWidth = 30
		cell.SetText(rules[r-1].Tags())
	case 5:
		maxWidth = 25
		cell.SetText(rules[r-1].Destination.Bucket)
	case 6:
		maxWidth = 10
		cell.SetText(rules[r-1].Destination.StorageClass)
	}
	cell.SetMaxWidth(maxWidth)
	return cell
}
