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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminTierListCmd = cli.Command{
	Name:         "ls",
	Usage:        "lists remote tier targets",
	Action:       mainAdminTierList,
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
  1. List remote tier targets configured in myminio
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminTierListSyntax - validate all the passed arguments
func checkAdminTierListSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr > 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-ls subcommand.")
	}
}

type tierRowHdr int

const (
	tierNameHdr tierRowHdr = iota
	tierTypeHdr
	tierEndpointHdr
	tierBucketHdr
	tierPrefixHdr
	tierRegionHdr
	tierStorageClassHdr
)

var tierRowNames = []string{
	"Name",
	"Type",
	"Endpoint",
	"Bucket",
	"Prefix",
	"Region",
	"Storage-Class",
}

var tierColorScheme = []*color.Color{
	color.New(color.FgYellow),
	color.New(color.FgCyan),
	color.New(color.FgGreen),
	color.New(color.FgHiWhite),
	color.New(color.FgHiWhite),
	color.New(color.FgHiWhite),
	color.New(color.FgCyan),
}

func storageClass(t *madmin.TierConfig) string {
	switch t.Type {
	case madmin.S3:
		return t.S3.StorageClass
	case madmin.Azure:
		return t.Azure.StorageClass
	case madmin.GCS:
		return t.GCS.StorageClass
	default:
		return ""
	}
}

type tierCfg struct {
	*madmin.TierConfig
}

func (tc *tierCfg) toRow(lengths []int) []string {
	row := make([]string, len(tierRowNames))
	row[tierNameHdr] = tc.Name
	row[tierTypeHdr] = tc.Type.String()
	row[tierEndpointHdr] = tc.Endpoint()
	row[tierBucketHdr] = tc.Bucket()
	row[tierPrefixHdr] = tc.Prefix()
	row[tierRegionHdr] = tc.Region()
	row[tierStorageClassHdr] = storageClass(tc.TierConfig)
	for i := range tierRowNames {
		if lengths[i] < len(row[i]) {
			lengths[i] = len(row[i])
		}
	}
	return row
}

// getTierListRowsAndCols returns a list of rows and a list of column header
// metadata like color theme and max cell length, given a list of tiers. Each
// row is represented by a list of cells in that row.
func getTierListRowsAndCols(tiers []*madmin.TierConfig) ([][]string, []Field) {
	rows := make([][]string, len(tiers))
	rows[0] = tierRowNames
	lengths := make([]int, len(rows[0]))
	for i := range lengths {
		lengths[i] = len(rows[0][i])
	}
	for _, tier := range tiers {
		tierCfg := tierCfg{tier}
		rows = append(rows, tierCfg.toRow(lengths))
	}
	// add 2 spaces to each column's max length to improve readability of
	// each cell
	cols := make([]Field, len(tierRowNames))
	for i, hdr := range rows[0] {
		cols[i] = Field{
			colorTheme: hdr,
			maxLen:     lengths[i] + 2,
		}
	}
	return rows, cols
}

type tierListMessage struct {
	Status  string               `json:"status"`
	Context *cli.Context         `json:"-"`
	Tiers   []*madmin.TierConfig `json:"tiers"`
}

// String method returns a tabular listing of remote tier configurations.
func (msg *tierListMessage) String() string {
	if len(msg.Tiers) == 0 {
		return "No remote tier has been configured"
	}

	const tableSeparator = "|"
	rows, cols := getTierListRowsAndCols(msg.Tiers)
	tbl := newPrettyTable(tableSeparator, cols...)
	var contents string
	for _, row := range rows {
		contents += fmt.Sprintf("%s\n", tbl.buildRow(row...))
	}
	return contents
}

// JSON method returns JSON encoding of msg.
func (msg *tierListMessage) JSON() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

func mainAdminTierList(ctx *cli.Context) error {
	checkAdminTierListSyntax(ctx)

	for i, color := range tierColorScheme {
		console.SetColor(tierRowNames[i], color)
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	tiers, err := client.ListTiers(globalContext)
	if err != nil {
		fatalIf(probe.NewError(err).Trace(args...), "Unable to list configured remote tier targets")
	}

	printMsg(&tierListMessage{
		Status:  "success",
		Context: ctx,
		Tiers:   tiers,
	})
	return nil
}
