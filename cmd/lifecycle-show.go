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
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

// TODO: The usage and examples will change as the command implementation evolves after feedback.

var ilmShowCmd = cli.Command{
	Name:   "show",
	Usage:  "Get Information bucket/object lifecycle management information",
	Action: mainLifecycleShow,
	Before: setGlobalsFromContext,
	Flags:  append(ilmShowFlags, globalFlags...),
}

var ilmShowFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "minimum",
		Usage: "Show Minimum fields",
	},
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "Show Expiration Info",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "Show Transition Info",
	},
}

func getShowOpts(ctx *cli.Context) showDetails {
	showOpts := showDetails{
		expiry:       ctx.Bool("expiry"),
		transition:   ctx.Bool("transition"),
		json:         ctx.Bool("json"),
		minimum:      ctx.Bool("minimum"),
		allAvailable: true,
	}
	if showOpts.minimum {
		showOpts.allAvailable = false
	} else if !showOpts.expiry && !showOpts.transition && !showOpts.json {
		showOpts.allAvailable = true
	} else {
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

func setColorScheme() {
	console.SetColor(fieldMainHeader, color.New(color.Bold, color.FgHiRed))
	console.SetColor(fieldThemeRow, color.New(color.FgHiWhite))
	console.SetColor(fieldThemeHeader, color.New(color.FgCyan))
	console.SetColor(fieldThemeTick, color.New(color.FgGreen))
	console.SetColor(fieldThemeExpiry, color.New(color.BlinkRapid, color.FgGreen))
	console.SetColor(fieldThemeResultSuccess, color.New(color.FgGreen, color.Bold))
}

// checkIlmShowSyntax - validate arguments passed by a user
func checkIlmShowSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelp(ctx, "")
		os.Exit(globalErrorExitStatus)
	}
}

func printIlmShow(info ilmResult, showOpts showDetails) {
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
func checkAddHeaderCell(headerFields *[]Field, headerArr *[]string, rowCheck map[string]int, cellInfo tableCellInfo) {
	if _, ok := rowCheck[cellInfo.labelKey]; ok {
		*headerFields = append(*headerFields, Field{ /*cellInfo.fieldTheme*/ "", cellInfo.columnWidth})
		*headerArr = append(*headerArr, getAlignedText(cellInfo.label, cellInfo.align, cellInfo.columnWidth))
	}
}

func checkAddTableCell(fieldArr *[]Field, rowArr *[]string, rowCheck map[string]int, cellInfo tableCellInfo) {
	if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
		(*fieldArr)[colIdx] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
		(*rowArr)[colIdx] = getAlignedText(cellInfo.label, cellInfo.align, cellInfo.columnWidth)
	}
}

func getColWidthTable() map[string]int {
	colWidth := make(map[string]int)

	colWidth[idLabel] = idWidth
	colWidth[prefixLabel] = prefixWidth
	colWidth[statusLabel] = statusWidth
	colWidth[expiryLabel] = expiryWidth
	colWidth[expiryDatesLabelKey] = expiryDatesWidth
	colWidth[transitionLabel] = transitionWidth
	colWidth[transitionDateLabel] = transitionDateWidth
	colWidth[storageClassLabel] = storageClassWidth
	colWidth[tagLabel] = tagWidth

	return colWidth
}
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
	//var newFields []Field
	//var newRows []string
	colWidth := getColWidthTable()

	if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
		(*fieldArr)[colIdx] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
		// (*rowArr)[colIdx] = getCentered(cellInfo.multLabels[0], cellInfo.columnWidth)
		(*rowArr)[colIdx] = getLeftAlgined(cellInfo.multLabels[0], cellInfo.columnWidth)
	}
	for index := 1; index < multLth; index++ {
		fields, _ := newFields[index-1]
		rows, _ := newRows[index-1]
		fields = make([]Field, len(rowCheck))
		rows = make([]string, len(rowCheck))
		for k, v := range rowCheck {
			if k == cellInfo.labelKey {
				// *newRows = append(*newRows, cellInfo.multLabels[index])
				// *newFields = append(*newFields, Field{cellInfo.fieldTheme, cellInfo.columnWidth})
				// fields = append(fields, Field{cellInfo.fieldTheme, cellInfo.columnWidth})
				// rows = append(rows, cellInfo.multLabels[index] /*getCentered(cellInfo.multLabels[index], cellInfo.columnWidth)*/)
				fields[v] = Field{cellInfo.fieldTheme, cellInfo.columnWidth}
				// rows[v] = getCentered(cellInfo.multLabels[index], cellInfo.columnWidth)
				rows[v] = getLeftAlgined(cellInfo.multLabels[index], cellInfo.columnWidth)
			} else {
				var width int
				var ok bool
				if width, ok = colWidth[k]; ok {
					width = 4
				}
				// rows = append(rows, getCentered(blankCell, width))
				rows[v] = getCentered(blankCell, width)
				// fields = append(fields, Field{cellInfo.fieldTheme, width})
				fields[v] = Field{cellInfo.fieldTheme, width}
			}
		}
		//console.Println(len(rows))
		newFields[index-1] = fields
		newRows[index-1] = rows
	}
}
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
		tableCellInfo{label: statusLabel, labelKey: statusLabel, fieldTheme: fieldThemeHeader, columnWidth: statusWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: expiryLabel, labelKey: expiryLabel, fieldTheme: fieldThemeHeader, columnWidth: expiryWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: expiryDatesLabel, labelKey: expiryDatesLabelKey, fieldTheme: fieldThemeHeader, columnWidth: expiryDatesWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: transitionLabel, labelKey: transitionLabel, fieldTheme: fieldThemeHeader, columnWidth: transitionWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: transitionDateLabel, labelKey: transitionDatesLabelKey, fieldTheme: fieldThemeHeader, columnWidth: transitionDateWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: storageClassLabel, labelKey: storageClassLabel, fieldTheme: fieldThemeHeader, columnWidth: storageClassWidth, align: centerAlign})
	checkAddHeaderCell(&rowHeadFields, &mainHeadArr, rowCheck,
		tableCellInfo{label: tagLabel, labelKey: tagLabel, fieldTheme: fieldThemeHeader, columnWidth: tagWidth, align: centerAlign})

	tbl := newPrettyTable(tableSeperator, rowHeadFields...)
	tb := &tbl
	row := tb.buildRow(mainHeadArr...)
	row = fmt.Sprintf("%s", row)
	row = console.Colorize(fieldThemeHeader, row)
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
	if rule.Status == statusLabel {
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

func getTags(rule lifecycleRule) string {
	var tagBfr bytes.Buffer
	tagArr := rule.TagFilters
	tagLth := len(rule.TagFilters)
	var tagCellArr []string
	if len(rule.TagFilters) == 0 && rule.RuleFilter != nil && rule.RuleFilter.And != nil {
		tagLth = len(rule.RuleFilter.And.Tags)
		tagArr = rule.RuleFilter.And.Tags
	}
	for tagIdx := 0; tagIdx < tagLth; tagIdx++ {
		tagBfr.WriteString(tagArr[tagIdx].Key + ":" + tagArr[tagIdx].Value)
		tagCellArr = append(tagCellArr, tagArr[tagIdx].Key+":"+tagArr[tagIdx].Value)
		if tagIdx < len(tagArr)-1 {
			tagBfr.WriteString(tagSeperator)
		}
	}
	tagLbl := tagBfr.String()
	if len(tagLbl) == 0 {
		tagLbl = blankCell
	}
	return tagLbl
}

func getTagArr(rule lifecycleRule) []string {
	//var tagBfr bytes.Buffer
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

func printIlmRows(tb *PrettyTable, rowCheck map[string]int, info ilmResult, showOpts showDetails) {
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
			tableCellInfo{label: getStatusTick(rule), labelKey: statusLabel, fieldTheme: fieldThemeRow, columnWidth: statusWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getExpiryTick(rule), labelKey: expiryLabel, fieldTheme: fieldThemeTick, columnWidth: expiryWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getExpiryDateVal(rule), labelKey: expiryDatesLabelKey, fieldTheme: fieldThemeRow, columnWidth: expiryWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getTransitionTick(rule), labelKey: transitionLabel, fieldTheme: fieldThemeTick, columnWidth: transitionWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getTransitionDate(rule), labelKey: transitionDatesLabelKey, fieldTheme: fieldThemeRow, columnWidth: transitionDateWidth, align: centerAlign})
		checkAddTableCell(&rowFields, &rowArr, rowCheck,
			tableCellInfo{label: getStorageClassName(rule), labelKey: storageClassLabel, fieldTheme: fieldThemeRow, columnWidth: storageClassWidth, align: centerAlign})
		// checkAddTableCell(&rowFields, &rowArr, rowCheck,
		//	tableCellInfo{label: getTags(rule), labelKey: tagLabel, fieldTheme: fieldThemeRow, columnWidth: tagWidth, align: centerAlign})
		var newRows map[int][]string
		var newFields map[int][]Field
		newRows = make(map[int][]string)
		newFields = make(map[int][]Field)
		checkAddTableCellRows(&rowFields, &rowArr, rowCheck,
			tableCellInfo{multLabels: getTagArr(rule), label: "", labelKey: tagLabel, fieldTheme: fieldThemeRow, columnWidth: tagWidth, align: leftAlign},
			newFields, newRows)
		var row string // Table row
		row = tb.buildRow(rowArr...)
		row = fmt.Sprintf("%s", row)
		console.Println(row)
		lineRow := buildLineRow(rowArr)
		if len(newRows) > 0 {
			for index := 0; index < len(newRows); index++ {
				newRow, ok := newRows[index]
				// console.Println(strconv.Itoa(index) + " " + strconv.FormatBool(ok) + " " + strconv.Itoa(len(newRows)))
				if ok {
					row = tb.buildRow(newRow...)
					row = fmt.Sprintf("%s", row)
					console.Println(row)
				}
			}
		}
		row = tb.buildRow(lineRow...)
		row = fmt.Sprintf("%s", row)
		console.Println(row)
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
	transitionSet := (rule.Transition != nil) &&
		((rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero()) ||
			rule.Transition.TransitionInDays > 0)
	transitionDetailsShow := (showOpts.allAvailable || showOpts.transition)
	return transitionSet && transitionDetailsShow
}

func getColumns(info ilmResult, rowCheck map[string]int, showOpts showDetails) {
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
		if !ok && rule.ID != "" {
			rowCheck[idLabel] = colIdx
		}
		_, ok = rowCheck[prefixLabel]
		prefixVal := getPrefixVal(rule)
		if !ok && prefixVal != "" {
			rowCheck[prefixLabel] = incColIdx()
		}
		_, ok = rowCheck[statusLabel]
		if !ok {
			rowCheck[statusLabel] = incColIdx()
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
		_, ok = rowCheck[storageClassLabel]
		if !ok && showTransitionDetails(rule, showOpts) {
			rowCheck[storageClassLabel] = incColIdx()
		}
		_, ok = rowCheck[tagLabel]
		tagSet := len(rule.TagFilters) > 0 || (rule.RuleFilter != nil && rule.RuleFilter.And != nil && (len(rule.RuleFilter.And.Tags) > 0))
		if !ok && tagSet {
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
	lfcInfoXML, err := getIlmInfo(objectURL)
	if err != nil {
		fmt.Println(err)
		return err.ToGoError()
	}
	if lfcInfoXML == "" {
		return nil
	}
	lfcInfo := ilmResult{}
	err2 := xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo)
	// console.Println(lfcInfo)
	if err2 != nil {
		fmt.Println(err2)
		return err2
	}
	showOpts := getShowOpts(ctx)

	printIlmShow(lfcInfo, showOpts)
	return nil
}
