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

package ilm

import (
	"strconv"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

const (
	// rule ID field column width in table output
	idColumnWidth int = 22
	// rule prefix field column width in table output
	prefixColumnWidth int = 16
	// StatusColumnWidth column width in table output
	statusColumnWidth int = 12
	// IlmDatesColumnWidth column width in table output
	ilmDatesColumnWidth int = 14
	// TagsColumnWidth column width in table output
	tagsColumnWidth int = 24
	// TypeColumnWidth column width in table output
	typeColumnWidth int = 12
	// VersionColumnWidth column width in table output
	versionColumnWidth int = 14
	// StorageClassColumnWidth column width in table output
	storageClassColumnWidth int = 18
)

const (
	leftAlign   int = 1
	centerAlign int = 2
	rightAlign  int = 3
)

// Labels used for display.
const (
	idLabel             string = "ID"
	prefixLabel         string = "Prefix"
	statusLabel         string = "Enabled "
	expiryLabel         string = "Expiry"
	lifecycleDatesLabel string = "Date/Days    "
	tagLabel            string = "Tags"
	transitionLabel     string = "Transition"
	storageClassLabel   string = "Tier  "
	typeLabel           string = "Type"
	versionLabel        string = "Version "
	unknownLabel        string = "Unknown "
)

// Keys to be used in map structure which stores the columns to be displayed.
const (
	statusLabelKey       string = "Enabled"
	storageClassLabelKey string = "Storage-Class"
	ilmDatesLabelKey     string = "ILM-Dates"
)

// Some cell values
const (
	tickCell      string = "✔ "
	crossTickCell string = "✗ "
	blankCell     string = " "
)

// Used in tags. Ex: --tags "key1=value1&key2=value2&key3=value3"
const (
	tagSeperator    string = "&"
	keyValSeperator string = "="
)

// Represents information going into a single cell in the table.
type tableCellInfo struct {
	label       string
	multLabels  []string
	labelKey    string
	columnWidth int
	align       int
}

// Determines what columns need to be shown
type showDetails struct {
	allAvailable bool
	expiry       bool
	transition   bool
}

// PopulateILMDataForDisplay based on showDetails determined by user input, populate the ILM display
// table with information. Table is constructed row-by-row. Headers are first, then the rest of the rows.
func PopulateILMDataForDisplay(ilmCfg *lifecycle.Configuration, rowCheck *map[string]int, alignedHdrLabels *[]string,
	cellDataNoTags *[][]string, cellDataWithTags *[][]string, tagRows *map[string][]string,
	showAll, showExpiry, showTransition bool,
) {
	// We need the different column headers and their respective column index
	// where they appear in a map data-structure format.
	// [Column Label] -> [Column Number]
	*rowCheck = make(map[string]int)
	// For rows with tags only tags are shown. Rest of the cells are empty (blanks in full cell length)
	*tagRows = make(map[string][]string)
	showOpts := showDetails{
		allAvailable: showAll,
		expiry:       showExpiry,
		transition:   showTransition,
	}
	getColumns(ilmCfg, *rowCheck, alignedHdrLabels, showOpts)
	getILMShowDataWithoutTags(cellDataNoTags, *rowCheck, ilmCfg, showOpts)
	getILMShowDataWithTags(cellDataWithTags, *tagRows, *rowCheck, ilmCfg, showOpts)
}

// Text inside the table cell
func getAlignedText(label string, align int, columnWidth int) string {
	cellLabel := blankCell
	switch align {
	case leftAlign:
		cellLabel = getLeftAligned(label, columnWidth)
	case centerAlign:
		cellLabel = getCenterAligned(label, columnWidth)
	case rightAlign:
		cellLabel = getRightAligned(label, columnWidth)
	}
	return cellLabel
}

// GetColumnWidthTable We will use this map of Header Labels -> Column width
func getILMColumnWidthTable() map[string]int {
	colWidth := make(map[string]int)

	colWidth[idLabel] = idColumnWidth
	colWidth[prefixLabel] = prefixColumnWidth
	colWidth[statusLabelKey] = statusColumnWidth
	colWidth[typeLabel] = typeColumnWidth
	colWidth[versionLabel] = versionColumnWidth
	colWidth[lifecycleDatesLabel] = ilmDatesColumnWidth
	colWidth[storageClassLabelKey] = storageClassColumnWidth
	colWidth[tagLabel] = tagsColumnWidth

	return colWidth
}

// checkAddTableCellRows multiple rows created by filling up each cell of the table.
// Multiple rows are required for display of data with tags.
// Each 'key:value' pair is shown in 1 row and the rest of it is cells populated with blanks.
func checkAddTableCellRows(rowArr *[]string, rowCheck map[string]int, showOpts showDetails,
	cellInfo tableCellInfo, ruleID string, newRows map[string][]string,
) {
	var cellLabel string
	multLth := len(cellInfo.multLabels)
	if cellInfo.label != "" || multLth == 0 {
		if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
			(*rowArr)[colIdx] = getCenterAligned(blankCell, cellInfo.columnWidth)
		}
		return
	}
	colWidth := getILMColumnWidthTable()

	if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
		cellLabel := cellInfo.multLabels[0]
		if len(cellInfo.multLabels[0]) > (cellInfo.columnWidth - 3) { // 2 dots & 1 space for left-alignment
			cellLabel = cellLabel[:(cellInfo.columnWidth-5)] + ".."
		}
		(*rowArr)[colIdx] = getLeftAligned(cellLabel, cellInfo.columnWidth)
	}

	for index := 1; index < multLth; index++ {
		row := make([]string, len(rowCheck))
		for k, v := range rowCheck {
			if k == cellInfo.labelKey {
				cellLabel = cellInfo.multLabels[index]
				if len(cellInfo.multLabels[index]) > (cellInfo.columnWidth - 3) {
					cellLabel = cellLabel[:(cellInfo.columnWidth-5)] + ".."
				}
				row[v] = getLeftAligned(cellLabel, cellInfo.columnWidth)
			} else {
				var width int
				var ok bool
				if width, ok = colWidth[k]; !ok {
					width = 4
				}
				row[v] = getCenterAligned(blankCell, width)
			}
		}
		newRows[ruleID+strconv.Itoa(index-1)] = row
	}
}

// The right kind of tick is returned. Cross-tick if status is 'Disabled' & tick if status is 'Enabled'.
func getStatusTick(rule lifecycle.Rule) string {
	statusTick := crossTickCell
	if rule.Status == statusLabelKey {
		statusTick = tickCell
	}
	return statusTick
}

// Expiry date. 'YYYY-MM-DD'. Set for 00:00:00 GMT as per the standard.
func getLifeCycleDateVal(rule lifecycle.Rule) string {
	columnDate := blankCell
	// For Expiration Type Rules
	if !rule.Expiration.IsDateNull() {
		columnDate = strconv.Itoa(rule.Expiration.Date.Day()) + " " +
			rule.Expiration.Date.Month().String()[0:3] + " " +
			strconv.Itoa(rule.Expiration.Date.Year())
	} else if !rule.Expiration.IsDaysNull() {
		columnDate = strconv.Itoa(int(rule.Expiration.Days)) + " day(s)"
	}

	if !rule.Expiration.IsDateNull() {
		columnDate = strconv.Itoa(rule.Expiration.Date.Day()) + " " +
			rule.Expiration.Date.Month().String()[0:3] + " " +
			strconv.Itoa(rule.Expiration.Date.Year())
	} else if !rule.Expiration.IsDaysNull() {
		columnDate = strconv.Itoa(int(rule.Expiration.Days)) + " day(s)"
	}

	if !rule.NoncurrentVersionExpiration.IsDaysNull() {
		columnDate = strconv.Itoa(int(rule.NoncurrentVersionExpiration.NoncurrentDays)) + " day(s)"
	}

	if !rule.Transition.IsDateNull() {
		columnDate = strconv.Itoa(rule.Transition.Date.Day()) + " " +
			rule.Transition.Date.Month().String()[0:3] + " " +
			strconv.Itoa(rule.Transition.Date.Year())
	} else if !rule.Transition.IsDaysNull() {
		columnDate = strconv.Itoa(int(rule.Transition.Days)) + " day(s)"
	}

	if !rule.NoncurrentVersionTransition.IsDaysNull() {
		columnDate = strconv.Itoa(int(rule.NoncurrentVersionTransition.NoncurrentDays)) + " day(s)"
	}

	return columnDate
}

// Returns whether the rule is Transition or Expiration
func getIlmRuleType(rule lifecycle.Rule) string {
	if !rule.Transition.IsNull() {
		return transitionLabel
	}
	if !rule.NoncurrentVersionTransition.IsDaysNull() {
		return transitionLabel
	}
	if !rule.Expiration.IsNull() {
		return expiryLabel
	}
	if !rule.NoncurrentVersionExpiration.IsDaysNull() {
		return expiryLabel
	}
	return unknownLabel
}

// Returns whether the rule is for current or non-current
func getTargetVersion(rule lifecycle.Rule) string {
	if !rule.NoncurrentVersionTransition.IsDaysNull() {
		return "Non-Current"
	}
	if !rule.Transition.IsNull() {
		return "Current"
	}
	if !rule.NoncurrentVersionExpiration.IsDaysNull() {
		return "Non-Current"
	}
	if !rule.Expiration.IsNull() {
		return "Current"
	}
	return unknownLabel
}

// Storage class name for transition.
func getStorageClassName(rule lifecycle.Rule) string {
	storageClass := blankCell
	transitionSet := !rule.Transition.IsNull()
	storageClassAvail := transitionSet && (rule.Transition.StorageClass != "")
	if storageClassAvail {
		storageClass = rule.Transition.StorageClass
	}
	return storageClass
}

// Array of Tag strings, each in key:value format
func getTagArr(rule lifecycle.Rule) []string {
	var tagArr []lifecycle.Tag
	switch {
	case !rule.RuleFilter.Tag.IsEmpty():
		tagArr = []lifecycle.Tag{rule.RuleFilter.Tag}
	case !rule.RuleFilter.And.IsEmpty():
		tagArr = rule.RuleFilter.And.Tags
	default:
		return []string{}
	}

	tagLth := len(tagArr)
	tagCellArr := make([]string, len(tagArr))
	for tagIdx := 0; tagIdx < tagLth; tagIdx++ {
		tagCellArr[tagIdx] = (tagArr[tagIdx].Key + ":" + tagArr[tagIdx].Value)
	}
	return tagCellArr
}

// Add single row table cell - non-header.
func checkAddTableCell(rowArr *[]string, rowCheck map[string]int, cellInfo tableCellInfo) {
	if rowArr == nil {
		return
	}
	if len(*rowArr) == 0 && len(rowCheck) > 0 {
		*rowArr = make([]string, len(rowCheck))
	}

	if colIdx, ok := rowCheck[cellInfo.labelKey]; ok {
		if len(cellInfo.label)%2 != 0 && len(cellInfo.label) < cellInfo.columnWidth {
			cellInfo.label += " "
		} else if len(cellInfo.label) > (cellInfo.columnWidth - 2) { // 2 dots to indicate text longer than column width
			cellInfo.label = cellInfo.label[:(cellInfo.columnWidth-6)] + ".."
		}

		(*rowArr)[colIdx] = getAlignedText(cellInfo.label, cellInfo.align, cellInfo.columnWidth)
	}
}

// GetILMShowDataWithoutTags - Without tags
func getILMShowDataWithoutTags(cellInfo *[][]string, rowCheck map[string]int, info *lifecycle.Configuration, showOpts showDetails) {
	*cellInfo = make([][]string, 0)
	count := 0
	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]

		// Ignore this rule if it has tag filter in it
		if len(getTagArr(rule)) > 0 {
			continue
		}

		showExpiry := !rule.Expiration.IsNull()
		transitionSet := !rule.Transition.IsNull()
		skipExpTran := (showOpts.expiry && !showExpiry) || (showOpts.transition && !transitionSet)
		if skipExpTran {
			continue
		}

		*cellInfo = append(*cellInfo, make([]string, 0))
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: rule.ID, labelKey: idLabel, columnWidth: idColumnWidth, align: leftAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getIlmRuleType(rule), labelKey: typeLabel, columnWidth: typeColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getTargetVersion(rule), labelKey: versionLabel, columnWidth: versionColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStorageClassName(rule), labelKey: storageClassLabelKey, columnWidth: storageClassColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getLifeCycleDateVal(rule), labelKey: ilmDatesLabelKey, columnWidth: ilmDatesColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getPrefixVal(rule), labelKey: prefixLabel, columnWidth: prefixColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStatusTick(rule), labelKey: statusLabelKey, columnWidth: statusColumnWidth, align: centerAlign})

		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: blankCell, labelKey: tagLabel, columnWidth: tagsColumnWidth, align: centerAlign})
		count++
	}
}

// GetILMShowDataWithTags Just the data with extra rows for extra tags
func getILMShowDataWithTags(cellInfo *[][]string, newRows map[string][]string, rowCheck map[string]int, info *lifecycle.Configuration, showOpts showDetails) {
	*cellInfo = make([][]string, 0)
	count := 0
	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]

		// Ignore printing this rule if it doesn't have any tag filter in it
		if len(getTagArr(rule)) == 0 {
			continue
		}

		showExpiry := !rule.Expiration.IsNull()
		transitionSet := !rule.Transition.IsNull()
		skipExpTran := (showOpts.expiry && !showExpiry) || (showOpts.transition && !transitionSet)
		if skipExpTran {
			continue
		}

		*cellInfo = append(*cellInfo, make([]string, 0))
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: rule.ID, labelKey: idLabel, columnWidth: idColumnWidth, align: leftAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getIlmRuleType(rule), labelKey: typeLabel, columnWidth: versionColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getTargetVersion(rule), labelKey: versionLabel, columnWidth: typeColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStorageClassName(rule), labelKey: storageClassLabelKey, columnWidth: storageClassColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getLifeCycleDateVal(rule), labelKey: ilmDatesLabelKey, columnWidth: ilmDatesColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getPrefixVal(rule), labelKey: prefixLabel, columnWidth: prefixColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStatusTick(rule), labelKey: statusLabelKey, columnWidth: statusColumnWidth, align: centerAlign})

		checkAddTableCellRows(&((*cellInfo)[count]), rowCheck, showOpts,
			tableCellInfo{multLabels: getTagArr(rule), label: "", labelKey: tagLabel, columnWidth: tagsColumnWidth, align: leftAlign},
			rule.ID, newRows)
		count++
	}
}

func getPrefixVal(rule lifecycle.Rule) string {
	prefixVal := ""
	switch {
	case rule.Prefix != "":
		prefixVal = rule.Prefix
	case !rule.RuleFilter.And.IsEmpty():
		prefixVal = rule.RuleFilter.And.Prefix
	case rule.RuleFilter.Prefix != "":
		prefixVal = rule.RuleFilter.Prefix
	}
	return prefixVal
}

func showLifeCycleDetails(rule lifecycle.Rule, showOpts showDetails) bool {
	if showOpts.allAvailable {
		return true
	}
	expirySet := !rule.Expiration.IsNull()

	return (expirySet && (showOpts.allAvailable || showOpts.expiry))
}

func showTags(rule lifecycle.Rule, showOpts showDetails) bool {
	tagSet := showOpts.allAvailable || !rule.RuleFilter.And.IsEmpty()
	return tagSet
}

func getColumns(info *lifecycle.Configuration, rowCheck map[string]int, alignedHdrLabels *[]string, showOpts showDetails) {
	tagIn := false // Keep tag in the end
	colIdx := 0
	colWidthTbl := getILMColumnWidthTable()
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
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(idLabel, centerAlign, colWidthTbl[idLabel]))
		}
		_, ok = rowCheck[typeLabel]
		if !ok {
			rowCheck[typeLabel] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(typeLabel, centerAlign, colWidthTbl[typeLabel]))
		}
		_, ok = rowCheck[versionLabel]
		if !ok {
			rowCheck[versionLabel] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(versionLabel, centerAlign, colWidthTbl[versionLabel]))
		}
		_, ok = rowCheck[storageClassLabelKey]
		if !ok {
			rowCheck[storageClassLabelKey] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(storageClassLabel, centerAlign, colWidthTbl[storageClassLabelKey]))
		}
		_, ok = rowCheck[ilmDatesLabelKey]
		if !ok && showLifeCycleDetails(rule, showOpts) {
			rowCheck[ilmDatesLabelKey] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(lifecycleDatesLabel, centerAlign, colWidthTbl[ilmDatesLabelKey]))
		}
		_, ok = rowCheck[prefixLabel]
		if !ok { // ID & Prefix are shown always.
			rowCheck[prefixLabel] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(prefixLabel, centerAlign, colWidthTbl[prefixLabel]))
		}
		_, ok = rowCheck[statusLabelKey]
		if !ok {
			rowCheck[statusLabelKey] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(statusLabel, centerAlign, colWidthTbl[statusLabelKey]))
		}
		_, ok = rowCheck[tagLabel]
		if !ok && showTags(rule, showOpts) {
			rowCheck[tagLabel] = incColIdx()
			tagIn = true
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(tagLabel, centerAlign, colWidthTbl[tagLabel]))
		}
	}
}
