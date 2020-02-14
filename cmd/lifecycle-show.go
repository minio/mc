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
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

var ilmShowCmd = cli.Command{
	Name:   "show",
	Usage:  "show bucket lifecycle configuration",
	Action: mainLifecycleShow,
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
  Ilm show command displays current lifecycle configuration.

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

var ilmShowFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "show expiration information",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "show transition information",
	},
	cli.BoolFlag{
		Name:  "minimum",
		Usage: "show minimum fields",
	},
}

// showopts gives an idea about what details user prefers to see.
func getShowOpts(ctx *cli.Context) showDetails {
	showOpts := showDetails{
		expiry:       ctx.Bool("expiry"),
		transition:   ctx.Bool("transition"),
		json:         ctx.Bool("json"),
		minimum:      ctx.Bool("minimum"),
		allAvailable: true,
	}
	if showOpts.minimum || showOpts.expiry || showOpts.transition || showOpts.json {
		showOpts.allAvailable = false
	}
	if showOpts.allAvailable {
		showOpts.expiry = false
		showOpts.transition = false
		showOpts.minimum = false
		showOpts.json = false
	}

	return showOpts
}

// Color scheme for the table
func setColorScheme() {
	console.SetColor(fieldMainHeader, color.New(color.Bold, color.FgHiRed))
	console.SetColor(fieldThemeRow, color.New(color.FgHiWhite))
	console.SetColor(fieldThemeHeader, color.New(color.FgCyan))
	console.SetColor(fieldThemeTick, color.New(color.FgGreen))
	console.SetColor(fieldThemeExpiry, color.New(color.BlinkRapid, color.FgGreen))
	console.SetColor(fieldThemeResultSuccess, color.New(color.FgGreen, color.Bold))
	console.SetColor(fieldThemeResultFailure, color.New(color.FgHiYellow, color.Bold))
}

// checkIlmShowSyntax - validate arguments passed by a user
func checkIlmShowSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelp(ctx, "show")
		os.Exit(globalErrorExitStatus)
	}
}

// lifecycleConfiguration instance has the rules. Based on showDetails mentioned by the user, show table with information.
// Table is constructed row-by-row.
func printIlmShow(info lifecycleConfiguration, showOpts showDetails) {
	// [Column Label] -> [Column Number]
	if showOpts.json {
		printIlmJSON(info)
		return
	}
	rowCheck := make(map[string]int)

	getColumns(info, rowCheck, showOpts)
	var tb *PrettyTable
	tb = printIlmHeader(rowCheck)
	printIlmRows(tb, rowCheck, info, showOpts)
}

// Text inside the table cell
func getAlignedText(label string, align int, columnWidth int) string {
	cellLabel := blankCell
	switch align {
	case leftAlign:
		cellLabel = getLeftAlgined(label, columnWidth)
	case centerAlign:
		cellLabel = getCentered(label, columnWidth)
	case rightAlign:
		cellLabel = getRightAligned(label, columnWidth)
	}
	return cellLabel
}

// Add single table cell - header.
func checkAddHeaderCell(headerFields *[]Field, headerArr *[]string, rowCheck map[string]int, cellInfo tableCellInfo) {
	if _, ok := rowCheck[cellInfo.labelKey]; ok {
		*headerFields = append(*headerFields, Field{"", cellInfo.columnWidth})
		*headerArr = append(*headerArr, getAlignedText(cellInfo.label, cellInfo.align, cellInfo.columnWidth))
	}
}

// Add single table cell - non-header.
func checkAddTableCell(fieldArr *[]Field, rowArr *[]string, rowCheck map[string]int, cellInfo tableCellInfo) {
	if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
		(*fieldArr)[colIdx] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
		if len(cellInfo.label)%2 != 0 && len(cellInfo.label) < cellInfo.columnWidth {
			cellInfo.label += " "
		}
		(*rowArr)[colIdx] = getAlignedText(cellInfo.label, cellInfo.align, cellInfo.columnWidth)
	}
}

// We will use this map of Header Labels -> Column width
func getColumnWidthTable() map[string]int {
	colWidth := make(map[string]int)

	colWidth[idLabel] = idWidth
	colWidth[prefixLabel] = prefixWidth
	colWidth[statusLabelKey] = statusWidth
	colWidth[expiryLabel] = expiryWidth
	colWidth[expiryDatesLabelKey] = expiryDatesWidth
	colWidth[transitionLabel] = transitionWidth
	colWidth[transitionDateLabel] = transitionDateWidth
	colWidth[storageClassLabelKey] = storageClassWidth
	colWidth[tagLabel] = tagWidth

	return colWidth
}

// This cell will have multiple rows.
// This function was first written for multiple tags as we couldn't have all tags in 1 row.
func checkAddTableCellRows(fieldArr *[]Field, rowArr *[]string, rowCheck map[string]int, cellInfo tableCellInfo,
	newFields map[int][]Field, newRows map[int][]string) {
	multLth := len(cellInfo.multLabels)
	if cellInfo.label != "" || multLth <= 0 {
		if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
			(*fieldArr)[colIdx] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
			(*rowArr)[colIdx] = getCentered(blankCell, cellInfo.columnWidth)
		}
		return
	}
	colWidth := getColumnWidthTable()

	if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
		(*fieldArr)[colIdx] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
		cellLabel := cellInfo.multLabels[0]
		if len(cellInfo.multLabels[0]) > cellInfo.columnWidth {
			cellLabel = cellLabel[:(cellInfo.columnWidth-5)] + ".."
		}
		(*rowArr)[colIdx] = getLeftAlgined(cellLabel, cellInfo.columnWidth)
	}
	for index := 1; index < multLth; index++ {
		fields := make([]Field, len(rowCheck))
		rows := make([]string, len(rowCheck))
		for k, v := range rowCheck {
			if k == cellInfo.labelKey {
				fields[v] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
				cellLabel := cellInfo.multLabels[index]
				if len(cellInfo.multLabels[index]) > cellInfo.columnWidth {
					cellLabel = cellLabel[:(cellInfo.columnWidth-5)] + ".."
				}
				rows[v] = getLeftAlgined(cellLabel, cellInfo.columnWidth)
			} else {
				var width int
				var ok bool
				if width, ok = colWidth[k]; ok {
					width = 4
				}
				rows[v] = getCentered(blankCell, width)
				fields[v] = Field{cellInfo.fieldTheme, width}
			}
		}
		newFields[index-1] = fields
		newRows[index-1] = rows
	}
}

// Header row is printed first as it could be a different color/shade
func printIlmHeader(rowCheck map[string]int) *PrettyTable {
	if len(rowCheck) <= 0 {
		return nil
	}
	mainHeadArr := []string{}
	rowHeadFields := []Field{}

	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: idLabel, labelKey: idLabel, fieldTheme: fieldThemeHeader, columnWidth: idWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: prefixLabel, labelKey: prefixLabel, fieldTheme: fieldThemeHeader, columnWidth: prefixWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: statusLabel, labelKey: statusLabelKey, fieldTheme: fieldThemeHeader, columnWidth: statusWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: expiryLabel, labelKey: expiryLabel, fieldTheme: fieldThemeHeader, columnWidth: expiryWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: expiryDatesLabel, labelKey: expiryDatesLabelKey, fieldTheme: fieldThemeHeader, columnWidth: expiryDatesWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: transitionLabel, labelKey: transitionLabel, fieldTheme: fieldThemeHeader, columnWidth: transitionWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: transitionDateLabel, labelKey: transitionDatesLabelKey, fieldTheme: fieldThemeHeader, columnWidth: transitionDateWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: storageClassLabel, labelKey: storageClassLabelKey, fieldTheme: fieldThemeHeader, columnWidth: storageClassWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: tagLabel, labelKey: tagLabel, fieldTheme: fieldThemeHeader, columnWidth: tagWidth, align: centerAlign})

	// This table is used throughout the configuration display.
	tbl := newPrettyTable(tableSeperator, rowHeadFields...)
	tb := &tbl
	row := tb.buildRow(mainHeadArr...)
	row = console.Colorize(fieldThemeHeader, row)
	console.Println(row)
	// Build & print line
	lineRow := buildLineRow(mainHeadArr)
	row = tb.buildRow(lineRow...)
	console.Println(row)
	return tb
}

func getExpiryTick(rule lifecycleRule) string {
	expiryTick := crossTickCell
	expiryDateSet := rule.Expiration != nil && rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()
	expirySet := rule.Expiration != nil && (expiryDateSet || rule.Expiration.ExpirationInDays > 0)
	if expirySet {
		expiryTick = tickCell
	}
	return expiryTick
}

func getStatusTick(rule lifecycleRule) string {
	statusTick := crossTickCell
	if rule.Status == statusLabelKey {
		statusTick = tickCell
	}
	return statusTick
}

func getExpiryDateVal(rule lifecycleRule) string {
	expiryDate := blankCell
	expirySet := (rule.Expiration != nil)
	expiryDateSet := expirySet && rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()
	if expiryDateSet {
		expiryDate = strconv.Itoa(rule.Expiration.ExpirationDate.Day()) + " " +
			rule.Expiration.ExpirationDate.Month().String()[0:3] + " " +
			strconv.Itoa(rule.Expiration.ExpirationDate.Year())
	} else if expirySet && rule.Expiration.ExpirationInDays > 0 {
		expiryDate = strconv.Itoa(rule.Expiration.ExpirationInDays) + " day(s)"
	}
	return expiryDate
}

func getTransitionTick(rule lifecycleRule) string {
	transitionSet := rule.Transition != nil
	transitionDateSet := transitionSet && ((rule.Transition.TransitionDate != nil &&
		!rule.Transition.TransitionDate.IsZero()) ||
		rule.Transition.TransitionInDays > 0)
	if !transitionSet || !transitionDateSet {
		return crossTickCell
	}
	return tickCell
}

func getTransitionDate(rule lifecycleRule) string {
	transitionDate := blankCell
	transitionSet := (rule.Transition != nil)
	transitionDateSet := transitionSet && (rule.Transition.TransitionDate != nil &&
		!rule.Transition.TransitionDate.IsZero())
	transitionDaySet := transitionSet && (rule.Transition.TransitionInDays > 0)
	if transitionDateSet {
		transitionDate = strconv.Itoa(rule.Transition.TransitionDate.Day()) + " " +
			rule.Transition.TransitionDate.Month().String()[0:3] + " " +
			strconv.Itoa(rule.Transition.TransitionDate.Year())
	} else if transitionDaySet {
		transitionDate = strconv.Itoa(rule.Transition.TransitionInDays) + " day(s)"
	}
	return transitionDate
}

func getStorageClassName(rule lifecycleRule) string {
	storageClass := blankCell
	transitionSet := (rule.Transition != nil)
	strgClsAval := transitionSet && (rule.Transition.StorageClass != "")
	if strgClsAval {
		storageClass = rule.Transition.StorageClass
	}
	return storageClass
}

// Array of Tag strings, each in key:value format
func getTagArr(rule lifecycleRule) []string {
	tagArr := rule.TagFilters
	tagLth := len(rule.TagFilters)
	if len(rule.TagFilters) == 0 && rule.RuleFilter != nil && rule.RuleFilter.And != nil {
		tagLth = len(rule.RuleFilter.And.Tags)
		tagArr = rule.RuleFilter.And.Tags
	}
	tagCellArr := make([]string, len(tagArr))
	for tagIdx := 0; tagIdx < tagLth; tagIdx++ {
		tagCellArr[tagIdx] = (tagArr[tagIdx].Key + ":" + tagArr[tagIdx].Value)
	}
	return tagCellArr
}

// Each lifeCycleRule in lifecycleConfiguration is printed one per row.
func printIlmRows(tb *PrettyTable, rowCheck map[string]int, info lifecycleConfiguration, showOpts showDetails) {
	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]
		rowArr := make([]string, len(rowCheck))
		rowFields := make([]Field, len(rowCheck))
		showExpiry := (rule.Expiration != nil) && ((rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()) ||
			rule.Expiration.ExpirationInDays > 0)
		transitionSet := (rule.Transition != nil) && ((rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero()) ||
			rule.Transition.TransitionInDays > 0)
		skipExpTran := (showOpts.expiry && !showExpiry) || (showOpts.transition && !transitionSet)
		if skipExpTran {
			continue
		}
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: rule.ID, labelKey: idLabel, fieldTheme: fieldThemeRow, columnWidth: idWidth, align: leftAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getPrefixVal(rule), labelKey: prefixLabel, fieldTheme: fieldThemeRow, columnWidth: prefixWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getStatusTick(rule), labelKey: statusLabelKey, fieldTheme: fieldThemeRow, columnWidth: statusWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getExpiryTick(rule), labelKey: expiryLabel, fieldTheme: fieldThemeExpiry, columnWidth: expiryWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getExpiryDateVal(rule), labelKey: expiryDatesLabelKey, fieldTheme: fieldThemeExpiry, columnWidth: expiryDatesWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getTransitionTick(rule), labelKey: transitionLabel, fieldTheme: fieldThemeTick, columnWidth: transitionWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getTransitionDate(rule), labelKey: transitionDatesLabelKey, fieldTheme: fieldThemeRow, columnWidth: transitionDateWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getStorageClassName(rule), labelKey: storageClassLabelKey, fieldTheme: fieldThemeRow, columnWidth: storageClassWidth, align: centerAlign})
		var newRows map[int][]string
		var newFields map[int][]Field
		newRows = make(map[int][]string)
		newFields = make(map[int][]Field)
		checkAddTableCellRows(&rowFields, &rowArr, rowCheck,
			tableCellInfo{multLabels: getTagArr(rule), label: "", labelKey: tagLabel, fieldTheme: fieldThemeRow, columnWidth: tagWidth, align: leftAlign},
			newFields, newRows)
		printRowAndLine(tb, rowArr, newRows)
	}
}

func buildLineRow(rowArr []string) []string {
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

// Prints a rule with 1 or more line. Then prints a line with '-' character. That way we identify each row in tabular display.
func printRowAndLine(tb *PrettyTable, rowArr []string, newRows map[int][]string) {
	var row string // Table row
	// Build & print row
	row = tb.buildRow(rowArr...)
	console.Println(row)
	if len(newRows) > 0 {
		for index := 0; index < len(newRows); index++ {
			newRow, ok := newRows[index]
			if ok {
				row = tb.buildRow(newRow...)
				console.Println(row)
			}
		}
	}
	// Build & print line
	lineRow := buildLineRow(rowArr)
	row = tb.buildRow(lineRow...)
	console.Println(row)
}

func getPrefixVal(rule lifecycleRule) string {
	prefixVal := ""
	switch {
	case rule.Prefix != "":
		prefixVal = rule.Prefix
	case rule.RuleFilter != nil && rule.RuleFilter.And != nil && rule.RuleFilter.And.Prefix != "":
		prefixVal = rule.RuleFilter.And.Prefix
	case rule.RuleFilter != nil && rule.RuleFilter.Prefix != "":
		prefixVal = rule.RuleFilter.Prefix
	}
	return prefixVal
}

func showExpiryDetails(rule lifecycleRule, showOpts showDetails) bool {
	if showOpts.allAvailable {
		return true
	}
	expirySet := (rule.Expiration != nil) &&
		((rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()) ||
			rule.Expiration.ExpirationInDays > 0)

	return (expirySet && (showOpts.allAvailable || showOpts.expiry))

}

func showExpTick(showOpts showDetails) bool {
	return showOpts.allAvailable || showOpts.minimum
}

func showTransitionTick(showOpts showDetails) bool {
	return (showOpts.allAvailable || showOpts.minimum)
}

func showTransitionDetails(rule lifecycleRule, showOpts showDetails) bool {
	if showOpts.allAvailable {
		return true
	}
	transitionSet := (rule.Transition != nil) &&
		((rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero()) ||
			rule.Transition.TransitionInDays > 0)
	transitionDetailsShow := (showOpts.allAvailable || showOpts.transition)
	return transitionSet && transitionDetailsShow
}

func showTags(rule lifecycleRule, showOpts showDetails) bool {
	if showOpts.minimum {
		return false
	}
	tagSet := showOpts.allAvailable ||
		(len(rule.TagFilters) > 0 || (rule.RuleFilter != nil && rule.RuleFilter.And != nil && (len(rule.RuleFilter.And.Tags) > 0)))
	return tagSet
}

func getColumns(info lifecycleConfiguration, rowCheck map[string]int, showOpts showDetails) {
	tagIn := false // Keep tag in the end
	colIdx := 0
	incColIdx := func() int {
		if tagIn {
			colIdx = rowCheck[tagLabel]
			rowCheck[tagLabel] = colIdx + 1
		} else {
			colIdx++
		}
		return colIdx
	}
	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]
		_, ok := rowCheck[idLabel]
		if !ok { // ID & Prefix are shown always.
			rowCheck[idLabel] = colIdx
		}
		_, ok = rowCheck[prefixLabel]
		if !ok { // ID & Prefix are shown always.
			rowCheck[prefixLabel] = incColIdx()
		}
		_, ok = rowCheck[statusLabelKey]
		if !ok {
			rowCheck[statusLabelKey] = incColIdx()
		}
		_, ok = rowCheck[expiryLabel]
		if !ok && showExpTick(showOpts) {
			rowCheck[expiryLabel] = incColIdx()
		}
		_, ok = rowCheck[expiryDatesLabelKey]
		if !ok && showExpiryDetails(rule, showOpts) {
			rowCheck[expiryDatesLabelKey] = incColIdx()
		}
		_, ok = rowCheck[transitionLabel]
		if !ok && showTransitionTick(showOpts) {
			rowCheck[transitionLabel] = incColIdx()
		}
		_, ok = rowCheck[transitionDatesLabelKey]
		if !ok && showTransitionDetails(rule, showOpts) {
			rowCheck[transitionDatesLabelKey] = incColIdx()
		}
		_, ok = rowCheck[storageClassLabelKey]
		if !ok && showTransitionDetails(rule, showOpts) {
			rowCheck[storageClassLabelKey] = incColIdx()
		}
		_, ok = rowCheck[tagLabel]
		if !ok && showTags(rule, showOpts) {
			rowCheck[tagLabel] = incColIdx()
			tagIn = true
		}
	}
}

func mainLifecycleShow(ctx *cli.Context) error {
	checkIlmShowSyntax(ctx)
	setColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)

	lfcInfo, err := getIlmConfig(objectURL)
	if err != nil {
		console.Errorln("Unable to show lifecycle configuration for " + objectURL + ". Error: " + err.Error())
		return err
	}
	showOpts := getShowOpts(ctx)

	printIlmShow(lfcInfo, showOpts)
	return nil
}
