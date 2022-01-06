// Copyright (c) 2015-2021 MinIO, Inc.
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
	"errors"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminTierInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "Displays per-tier statistics of all tier targets",
	Action:       mainAdminTierInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Prints per-tier statistics of all remote tier targets configured in myminio
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminTierInfoSyntax - validate all the passed arguments
func checkAdminTierInfoSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr > 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-info subcommand.")
	}
}

type tierInfoRowHdr int

const (
	tierInfoNameHdr tierInfoRowHdr = iota
	tierInfoAPIHdr
	tierInfoTypeHdr
	tierInfoUsageHdr
	tierInfoObjectsHdr
	tierInfoVersionsHdr
)

var tierInfoRowNames = []string{
	"Tier Name",
	"API",
	"Type",
	"Usage",
	"Objects",
	"Versions",
}

var tierInfoColorScheme = []*color.Color{
	color.New(color.FgYellow),
	color.New(color.FgCyan),
	color.New(color.FgCyan),
	color.New(color.FgHiWhite),
	color.New(color.FgHiWhite),
	color.New(color.FgHiWhite),
}

type tierInfos []madmin.TierInfo

func (t tierInfos) NumRows() int {
	return len([]madmin.TierInfo(t))
}

func (t tierInfos) NumCols() int {
	return len(tierInfoRowNames)
}

func (t tierInfos) EmptyMessage() string {
	return "No remote tiers configured."
}

func (t tierInfos) MarshalJSON() ([]byte, error) {
	type tierInfo struct {
		Name  string
		API   string
		Type  string
		Stats madmin.TierStats
	}
	ts := make([]tierInfo, 0, len(t))
	for _, tInfo := range t {
		ts = append(ts, tierInfo{
			Name:  tInfo.Name,
			API:   tierInfoAPI(tInfo.Type),
			Type:  tierInfoType(tInfo.Type),
			Stats: tInfo.Stats,
		})
	}
	return json.Marshal(ts)
}

func tierInfoAPI(tierType string) string {
	switch tierType {
	case madmin.S3.String(), madmin.GCS.String():
		return tierType
	case madmin.Azure.String():
		return "blob"
	case "internal":
		return madmin.S3.String()
	default:
		return "unknown"
	}
}

func tierInfoType(tierType string) string {
	if tierType == "internal" {
		return "hot"
	}
	return "warm"
}

func (t tierInfos) ToRow(i int, ls []int) []string {
	row := make([]string, len(tierInfoRowNames))
	if i == -1 {
		copy(row, tierInfoRowNames)
	} else {
		tierInfo := t[i]
		row[tierInfoNameHdr] = tierInfo.Name
		row[tierInfoAPIHdr] = tierInfoAPI(tierInfo.Type)
		row[tierInfoTypeHdr] = tierInfoType(tierInfo.Type)
		row[tierInfoUsageHdr] = humanize.IBytes(tierInfo.Stats.TotalSize)
		row[tierInfoObjectsHdr] = strconv.Itoa(tierInfo.Stats.NumObjects)
		row[tierInfoVersionsHdr] = strconv.Itoa(tierInfo.Stats.NumVersions)
	}

	// update ls to accommodate this row's values
	for i := range tierInfoRowNames {
		if ls[i] < len(row[i]) {
			ls[i] = len(row[i])
		}
	}
	return row
}

func mainAdminTierInfo(ctx *cli.Context) error {
	checkAdminTierInfoSyntax(ctx)

	for i, color := range tierInfoColorScheme {
		console.SetColor(tierInfoRowNames[i], color)
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	var msg tierInfoMessage
	tInfos, err := client.TierStats(globalContext)
	if err != nil {
		msg = tierInfoMessage{
			Status:  "error",
			Context: ctx,
			Error:   err.Error(),
		}
	} else {
		msg = tierInfoMessage{
			Status:    "success",
			Context:   ctx,
			TierInfos: tierInfos(tInfos),
		}
	}
	printMsg(&msg)
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
	if msg.Status == "error" {
		fatal(probe.NewError(errors.New(msg.Error)), "Unable to get tier statistics")
	}
	return toTable(tierInfos(msg.TierInfos))
}

// JSON method returns JSON encoding of msg.
func (msg *tierInfoMessage) JSON() string {
	b, _ := json.Marshal(msg)
	return string(b)
}
