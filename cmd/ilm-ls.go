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
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/cmd/ilm"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/pkg/console"
)

var ilmListFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "display only expiration fields",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "display only transition fields",
	},
}

var ilmLsCmd = cli.Command{
	Name:         "ls",
	Usage:        "lists lifecycle configuration rules set on a bucket",
	Action:       mainILMList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(ilmListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  List lifecycle configuration rules set on a bucket.

EXAMPLES:
  1. List the lifecycle management rules (all fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. List the lifecycle management rules (expration date/days fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --expiry myminio/mybucket

  3. List the lifecycle management rules (transition date/days, storage class fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --transition myminio/mybucket

  4. List the lifecycle management rules in JSON format for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --json myminio/mybucket
`,
}

type ilmListMessage struct {
	Status  string                   `json:"status"`
	Target  string                   `json:"target"`
	Context *cli.Context             `json:"-"`
	Config  *lifecycle.Configuration `json:"config"`
}

func (i ilmListMessage) String() string {
	showExpiry := i.Context.Bool("expiry")
	showTransition := i.Context.Bool("transition")

	// If none of the flags are explicitly mentioned, all fields are shown.
	showAll := !showExpiry && !showTransition

	var hdrLabelIndexMap map[string]int
	var alignedHdrLabels []string
	var cellDataNoTags [][]string
	var cellDataWithTags [][]string
	var tagRows map[string][]string
	var tbl PrettyTable

	ilm.PopulateILMDataForDisplay(i.Config, &hdrLabelIndexMap, &alignedHdrLabels,
		&cellDataNoTags, &cellDataWithTags, &tagRows,
		showAll, showExpiry, showTransition)

	// Entire table content.
	var tblContents string

	// Fill up fields
	var fields []Field

	// The header table
	for _, hdr := range alignedHdrLabels {
		fields = append(fields, Field{ilmThemeHeader, len(hdr)})
	}

	tbl = newPrettyTable(tableSeperator, fields...)
	tblContents = getILMHeader(&tbl, alignedHdrLabels...)

	// Reuse the fields
	fields = nil

	// The data table
	var tblRowField *[]string
	if len(cellDataNoTags) == 0 {
		tblRowField = &cellDataWithTags[0]
	} else {
		tblRowField = &cellDataNoTags[0]
	}

	for _, hdr := range *tblRowField {
		fields = append(fields, Field{ilmThemeRow, len(hdr)})
	}

	tbl = newPrettyTable(tableSeperator, fields...)
	tblContents += getILMRowsNoTags(&tbl, &cellDataNoTags)
	tblContents += getILMRowsWithTags(&tbl, &cellDataWithTags, tagRows)

	return tblContents
}

func (i ilmListMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// validateILMListFlagSet - Only one of these flags needs to be set for display: --json, --expiry, --transition
func validateILMListFlagSet(ctx *cli.Context) bool {
	var flags = [...]bool{ctx.Bool("expiry"), ctx.Bool("transition"), ctx.Bool("json")}
	found := false
	for _, flag := range flags {
		if found && flag {
			return false
		} else if flag {
			found = true
		}
	}
	return true
}

// checkILMListSyntax - validate arguments passed by a user
func checkILMListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "ls", globalErrorExitStatus)
	}

	if !validateILMListFlagSet(ctx) {
		fatalIf(errInvalidArgument(), "only one display field flag is allowed per ls command. Refer mc "+ctx.Command.FullName()+" --help.")
	}
}

const tableSeperator = "|"

func getILMHeader(tbl *PrettyTable, alignedHdrLabels ...string) string {
	if len(alignedHdrLabels) == 0 {
		return ""
	}
	row := tbl.buildRow(alignedHdrLabels...)
	header := console.Colorize(ilmThemeHeader, row+"\n")
	lineRow := buildILMTableLineRow(alignedHdrLabels...)
	row = tbl.buildRow(lineRow...)
	row = console.Colorize(ilmThemeHeader, row+"\n")
	header += row
	return header
}

func buildILMTableLineRow(rowArr ...string) []string {
	lineRowArr := make([]string, len(rowArr))
	for index := 0; index < len(rowArr); index++ {
		var tagBfr bytes.Buffer
		for rowArrChars := 0; rowArrChars < len(rowArr[index]); rowArrChars++ {
			tagBfr.WriteByte('-')
		}
		lineRowArr[index] = tagBfr.String()
	}
	return lineRowArr
}

func getILMRowsNoTags(tbl *PrettyTable, cellDataNoTags *[][]string) string {
	if cellDataNoTags == nil || len(*cellDataNoTags) == 0 {
		return ""
	}
	var rows string
	for _, rowArr := range *cellDataNoTags {
		var row string // Table row
		// Build & print row
		row = tbl.buildRow(rowArr...)
		row = console.Colorize(ilmThemeRow, row)
		rows += row + "\n"
		lineRow := buildILMTableLineRow(rowArr...)
		row = tbl.buildRow(lineRow...)
		row = console.Colorize(ilmThemeRow, row)
		rows += row + "\n"
	}
	return rows
}

func getILMRowsWithTags(tbl *PrettyTable, cellDataWithTags *[][]string, newRows map[string][]string) string {
	if cellDataWithTags == nil || len(*cellDataWithTags) == 0 {
		return ""
	}
	var rows string
	for _, rowArr := range *cellDataWithTags {
		if rowArr == nil {
			continue
		}
		var row string // Table row
		// Build & print row
		row = tbl.buildRow(rowArr...)
		row = console.Colorize(ilmThemeRow, row)
		rows += row + "\n"
		// Add the extra blank rows & tag value in the right column
		if len(newRows) > 0 {
			for index := 0; index < len(newRows); index++ {
				newRow, ok := newRows[strings.TrimSpace(rowArr[0])+strconv.Itoa(index)]
				if ok {
					row = tbl.buildRow(newRow...)
					row = console.Colorize(ilmThemeRow, row)
					rows += row + "\n"
				}
			}
		}
		// Build & print the line row
		lineRow := buildILMTableLineRow(rowArr...)
		row = tbl.buildRow(lineRow...)
		row = console.Colorize(ilmThemeRow, row)
		rows += row + "\n"
	}
	return rows
}

func mainILMList(cliCtx *cli.Context) error {
	ctx, cancelILMList := context.WithCancel(globalContext)
	defer cancelILMList()

	checkILMListSyntax(cliCtx)
	setILMDisplayColorScheme()

	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	ilmCfg, err := client.GetLifecycle(ctx)
	fatalIf(err.Trace(args...), "Unable to get lifecycle")

	if len(ilmCfg.Rules) == 0 {
		fatalIf(probe.NewError(errors.New("lifecycle configuration not set")).Trace(urlStr),
			"Unable to ls lifecycle configuration")
	}

	printMsg(ilmListMessage{
		Status:  "success",
		Target:  urlStr,
		Context: cliCtx,
		Config:  ilmCfg,
	})

	return nil
}
