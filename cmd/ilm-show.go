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
	ilm "github.com/minio/mc/cmd/ilm"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmShowFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "show expiration fields",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "show transition fields",
	},
	cli.BoolFlag{
		Name:  "minimum",
		Usage: "show minimum fields",
	},
}

var ilmShowCmd = cli.Command{
	Name:   "show",
	Usage:  "show bucket lifecycle configuration",
	Action: mainILMShow,
	Before: setGlobalsFromContext,
	Flags:  append(ilmShowFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  ILM show command displays current lifecycle configuration.

EXAMPLES:
  1. Show the lifecycle management rules (all fields) for the testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} s3/testbucket

  2. Show the lifecycle management rules (fields related to expration date/days) for the testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --expiry s3/testbucket

  3. Show the lifecycle management rules (fields related to transition date/days, storage class) for the testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --transition s3/testbucket

  4. Show the lifecycle management rules (minimum details) for the testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --minimum s3/testbucket

  5. Show the lifecycle management rules in JSON format for the testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} --json s3/testbucket

`,
}

func invalidILMShowFlagSet(ctx *cli.Context) bool {
	var flags = [...]bool{ctx.Bool("expiry"), ctx.Bool("transition"), ctx.Bool("json"),
		ctx.Bool("minimum")}
	foundSet := false
	idx := 0
	for range flags {
		if foundSet && flags[idx] {
			return true
		} else if flags[idx] {
			foundSet = true
		}
		idx++
	}
	return false
}

// checkILMShowSyntax - validate arguments passed by a user
func checkILMShowSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelp(ctx, "show")
		os.Exit(globalErrorExitStatus)
	}
	if invalidILMShowFlagSet(ctx) {
		fatalIf(probe.NewError(errors.New("Invalid input flag(s)")), "Only one show field flag allowed for show command. Refer mc "+ctx.Command.FullName()+" --help.")
	}
}

const tableSeperator = "|"

func showILMHeader(tbl *PrettyTable, alignedHdrLabels ...string) {
	if len(alignedHdrLabels) == 0 {
		return
	}
	row := tbl.buildRow(alignedHdrLabels...)
	console.Println(row)
	lineRow := buildILMTableLineRow(alignedHdrLabels...)
	row = tbl.buildRow(lineRow...)
	row = console.Colorize(fieldThemeHeader, row)
	console.Println(row)
}

func buildILMTableLineRow(rowArr ...string) []string {
	lineRowArr := make([]string, len(rowArr))
	for index := 0; index < len(rowArr); index++ {
		var tagBfr bytes.Buffer
		for slth := 0; slth < len(rowArr[index]); slth++ {
			tagBfr.WriteByte('-')
		}
		lineRowArr[index] = tagBfr.String()
	}
	return lineRowArr
}

func showILMRowsNoTags(tbl *PrettyTable, cellDataNoTags *[][]string) {
	if cellDataNoTags == nil || len(*cellDataNoTags) == 0 {
		return
	}
	for _, rowArr := range *cellDataNoTags {
		var row string // Table row
		// Build & print row
		row = tbl.buildRow(rowArr...)
		row = console.Colorize(fieldThemeRow, row)
		console.Println(row)
		lineRow := buildILMTableLineRow(rowArr...)
		row = tbl.buildRow(lineRow...)
		row = console.Colorize(fieldThemeRow, row)
		console.Println(row)
	}
}

func showILMRowsWithTags(tbl *PrettyTable, cellDataWithTags *[][]string, newRows map[string][]string) {
	if cellDataWithTags == nil || len(*cellDataWithTags) == 0 {
		return
	}
	for _, rowArr := range *cellDataWithTags {
		if rowArr == nil {
			continue
		}
		var row string // Table row
		// Build & print row
		row = tbl.buildRow(rowArr...)
		row = console.Colorize(fieldThemeRow, row)
		console.Println(row)
		// Add the extra blank rows & tag value in the right column
		if len(newRows) > 0 {
			for index := 0; index < len(newRows); index++ {
				newRow, ok := newRows[strings.TrimSpace(rowArr[0])+strconv.Itoa(index)]
				if ok {
					row = tbl.buildRow(newRow...)
					row = console.Colorize(fieldThemeRow, row)
					console.Println(row)
				}
			}
		}
		// Build & print the line row
		lineRow := buildILMTableLineRow(rowArr...)
		row = tbl.buildRow(lineRow...)
		row = console.Colorize(fieldThemeRow, row)
		console.Println(row)
	}
}

func mainILMShow(ctx *cli.Context) error {
	checkILMShowSyntax(ctx)
	setILMDisplayColorScheme()
	var err error
	var hdrLabelIndexMap map[string]int
	var alignedHdrLabels []string
	var cellDataNoTags [][]string
	var cellDataWithTags [][]string
	var tagRows map[string][]string
	var lfcInfo string
	var tbl PrettyTable
	args := ctx.Args()
	objectURL := args.Get(0)

	if lfcInfo, err = getILMXML(objectURL); err != nil {
		console.Errorln("Unable to show lifecycle configuration for " + objectURL + ". Error: " + err.Error())
		return err
	}

	if globalJSON {
		ilm.DisplayILMJSON(lfcInfo)
		return nil
	}
	showExpiry := ctx.Bool("expiry")
	showTransition := ctx.Bool("transition")
	showMinimum := ctx.Bool("minimum")

	// If none of the flags are explicitly mentioned, all fields are shown.
	showAll := !showExpiry && !showTransition && !showMinimum

	if err = ilm.GetILMDataForShow(lfcInfo, &hdrLabelIndexMap, &alignedHdrLabels,
		&cellDataNoTags, &cellDataWithTags, &tagRows,
		showAll, showMinimum, showExpiry, showTransition); err != nil {
		console.Errorln(err.Error() + ". Error getting tabular data for ILM configuration.")
		return err
	}
	if len(cellDataNoTags) == 0 && len(cellDataWithTags) == 0 {
		return nil
	}

	// The header table
	var fields []Field
	// Fill up fields
	for _, hdr := range alignedHdrLabels {
		fields = append(fields, Field{fieldThemeHeader, len(hdr)})
	}
	tbl = newPrettyTable(tableSeperator, fields...)
	showILMHeader(&tbl, alignedHdrLabels...)
	fields = nil
	// The data table
	var tblRowField *[]string
	if len(cellDataNoTags) == 0 {
		tblRowField = &cellDataWithTags[0]
	} else {
		tblRowField = &cellDataNoTags[0]
	}
	for _, hdr := range *tblRowField {
		fields = append(fields, Field{fieldThemeRow, len(hdr)})
	}
	tbl = newPrettyTable(tableSeperator, fields...)
	showILMRowsNoTags(&tbl, &cellDataNoTags)
	showILMRowsWithTags(&tbl, &cellDataWithTags, tagRows)

	return nil
}
