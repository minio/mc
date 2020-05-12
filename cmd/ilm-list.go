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
	"bytes"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/cmd/ilm"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmListFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "show only expiration fields",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "show only transition fields",
	},
	cli.BoolFlag{
		Name:  "minimum",
		Usage: "show minimum fields - id, prefix, status, transition set, expiry set",
	},
}

var ilmListCmd = cli.Command{
	Name:   "list",
	Usage:  "list bucket lifecycle configuration",
	Action: mainILMList,
	Before: setGlobalsFromContext,
	Flags:  append(ilmListFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  ILM list command displays current lifecycle configuration.

EXAMPLES:
  1. List the lifecycle management rules (all fields) for testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} s3/testbucket

  2. List the lifecycle management rules (expration date/days fields) for testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --expiry s3/testbucket

  3. List the lifecycle management rules (transition date/days, storage class fields) for testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --transition s3/testbucket

  4. List the lifecycle management rules (minimum details) for testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --minimum s3/testbucket

  5. List the lifecycle management rules in JSON format for testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --json s3/testbucket

`,
}

type ilmListMessage struct {
	Status    string                     `json:"status"`
	Target    string                     `json:"target"`
	ILMConfig string                     `json:"-"`
	ILM       ilm.LifecycleConfiguration `json:"ilm"`
}

func (i ilmListMessage) String() string {
	if i.ILMConfig == "" {
		return console.Colorize(ilmThemeResultFailure, "Lifecycle configuration for `"+i.Target+"` not set.")
	}
	return i.ILMConfig
}

func (i ilmListMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// validateILMListFlagSet - Only one of these flags needs to be set for display: --json, --expiry, --transition, --minimum
func validateILMListFlagSet(ctx *cli.Context) bool {
	var flags = [...]bool{ctx.Bool("expiry"), ctx.Bool("transition"), ctx.Bool("json"),
		ctx.Bool("minimum")}
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
		cli.ShowCommandHelp(ctx, "list")
		os.Exit(globalErrorExitStatus)
	}
	if !validateILMListFlagSet(ctx) {
		fatalIf(probe.NewError(errors.New("Invalid input flag(s)")), "Only one show field flag allowed for list command. Refer mc "+ctx.Command.FullName()+" --help.")
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

func mainILMList(ctx *cli.Context) error {
	checkILMListSyntax(ctx)
	setILMDisplayColorScheme()
	var err error
	var hdrLabelIndexMap map[string]int
	var alignedHdrLabels []string
	var cellDataNoTags [][]string
	var cellDataWithTags [][]string
	var tagRows map[string][]string
	var tbl PrettyTable
	args := ctx.Args()
	objectURL := args.Get(0)
	lfcInfo, pErr := getBucketILMConfiguration(objectURL)
	fatalIf(pErr.Trace(objectURL), "Failed to list lifecycle configuration for "+objectURL)
	showExpiry := ctx.Bool("expiry")
	showTransition := ctx.Bool("transition")
	showMinimum := ctx.Bool("minimum")
	// If none of the flags are explicitly mentioned, all fields are shown.
	showAll := !showExpiry && !showTransition && !showMinimum

	err = ilm.GetILMDataForShow(lfcInfo, &hdrLabelIndexMap, &alignedHdrLabels,
		&cellDataNoTags, &cellDataWithTags, &tagRows,
		showAll, showMinimum, showExpiry, showTransition)
	fatalIf(probe.NewError(err), "Error getting tabular data for ILM configuration.")
	// The header table
	var fields []Field
	// Fill up fields
	var tblContents string
	var ilmConfig ilm.LifecycleConfiguration
	if len(cellDataNoTags) == 0 && len(cellDataWithTags) == 0 {
		dataStr := ""
		if showTransition {
			dataStr = "Transition "
		} else if showExpiry {
			dataStr = "Expiry "
		}
		fatalIf(probe.NewError(errors.New(dataStr+"Lifecycle configuration not set")),
			"Failed to list lifecycle configuration for "+objectURL+".")
	}

	if !globalJSON {
		for _, hdr := range alignedHdrLabels {
			fields = append(fields, Field{ilmThemeHeader, len(hdr)})
		}
		tbl = newPrettyTable(tableSeperator, fields...)
		tblContents = getILMHeader(&tbl, alignedHdrLabels...)
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

	} else {
		ilmConfig, err = ilm.GetILMConfig(lfcInfo)
		fatalIf(probe.NewError(err), "Failed to get lifecycle configuration for "+objectURL+".")
	}
	printMsg(ilmListMessage{
		Status:    "success",
		Target:    objectURL,
		ILMConfig: tblContents,
		ILM:       ilmConfig,
	})

	return nil
}
