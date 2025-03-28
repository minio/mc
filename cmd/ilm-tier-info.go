// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminTierInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display tier statistics",
	Action:       mainAdminTierInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS [NAME]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Prints per-tier statistics of all remote tier targets configured on 'myminio':
     {{.Prompt}} {{.HelpName}} myminio

  2. Print per-tier statistics of given tier name 'MINIOTIER-1':
     {{.Prompt}} {{.HelpName}} myminio MINIOTIER-1
`,
}

// checkAdminTierInfoSyntax - validate all the passed arguments
func checkAdminTierInfoSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	if argsNr == 2 && globalJSON {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-info subcommand with json output.")
	}
	if argsNr > 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-info subcommand.")
	}
}

type tierInfos []madmin.TierInfo

var _ table.Data = tierInfos(nil)

func (t tierInfos) At(row, col int) string {
	cell := "-"
	switch col {
	case 0:
		cell = t[row].Name
	case 1:
		cell = t[row].Type
	case 2:
		cell = tierInfoType(t[row].Type)
	case 3:
		cell = humanize.IBytes(t[row].Stats.TotalSize)
	case 4:
		cell = strconv.Itoa(t[row].Stats.NumObjects)
	case 5:
		cell = strconv.Itoa(t[row].Stats.NumVersions)
	}
	return cell
}

func (t tierInfos) Rows() int {
	return len(t)
}

func (t tierInfos) Columns() int {
	return len(t.Headers())
}

func (t tierInfos) Headers() []string {
	return []string{
		"Tier Name",
		"API",
		"Type",
		"Usage",
		"Objects",
		"Versions",
	}
}

func (t tierInfos) MarshalJSON() ([]byte, error) {
	type tierInfo struct {
		Name       string
		API        string
		Type       string
		Stats      madmin.TierStats
		DailyStats madmin.DailyTierStats
	}
	ts := make([]tierInfo, 0, len(t))
	for _, tInfo := range t {
		ts = append(ts, tierInfo{
			Name:       tInfo.Name,
			API:        tInfo.Type,
			Type:       tierInfoType(tInfo.Type),
			Stats:      tInfo.Stats,
			DailyStats: tInfo.DailyStats,
		})
	}
	return json.Marshal(ts)
}

func tierInfoType(tierType string) string {
	if tierType == "internal" {
		return "hot"
	}
	return "warm"
}

func mainAdminTierInfo(ctx *cli.Context) error {
	checkAdminTierInfoSyntax(ctx)
	args := ctx.Args()
	aliasedURL := args.Get(0)
	tier := args.Get(1)

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	var msg tierInfoMessage
	tInfos, e := client.TierStats(globalContext)
	if e != nil {
		msg = tierInfoMessage{
			Status:  "error",
			Context: ctx,
			Error:   e.Error(),
		}
	} else {
		msg = tierInfoMessage{
			Status:    "success",
			Context:   ctx,
			TierInfos: tierInfos(tInfos),
		}
	}

	if globalJSON {
		printMsg(&msg)
		return nil
	}

	var (
		HeaderStyle  = lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)
		EvenRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Align(lipgloss.Center)
		OddRowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Align(lipgloss.Center)
		NumbersStyle = lipgloss.NewStyle().Align(lipgloss.Right)
	)
	tableData := tierInfos(tInfos)
	var filteredData table.Data
	filteredData = table.NewFilter(tableData).
		Filter(func(row int) bool {
			if tier == "" {
				return true
			}
			return tableData.At(row, 0) == tier
		})

	if filteredData.Rows() == 0 {
		// check if that tier name is valid
		// if valid will show that with empty data
		tiers, e := client.ListTiers(globalContext)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to list configured remote tier targets")
		for _, t := range tiers {
			if t.Name == tier {
				filteredData = tierInfos([]madmin.TierInfo{
					{
						Name: tier,
						Type: t.Type.String(),
					},
				})
				break
			}
		}
	}

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		Headers(tableData.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style
			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = EvenRowStyle
			default:
				style = OddRowStyle
			}
			switch col {
			case 3, 4, 5:
				style = NumbersStyle.Foreground(style.GetForeground())
			}
			return style
		}).
		Data(filteredData)

	if filteredData.Rows() == 0 {
		if tier != "" {
			console.Printf("No remote tiers' name match %s\n", tier)
		} else {
			console.Println("No remote tiers configured")
		}
		return nil
	}
	fmt.Println(tbl)

	return nil
}

type tierInfoMessage struct {
	Status    string       `json:"status"`
	Context   *cli.Context `json:"-"`
	TierInfos tierInfos    `json:"tiers,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// String method returns a tabular listing of remote tier configurations.
func (msg *tierInfoMessage) String() string {
	return "" // Not used, present to satisfy msg interface
}

// JSON method returns JSON encoding of msg.
func (msg *tierInfoMessage) JSON() string {
	b, _ := json.Marshal(msg)
	return string(b)
}
