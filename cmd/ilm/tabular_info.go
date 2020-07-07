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

package ilm

import (
	"strconv"
)

const (
	// rule ID field column width in table output
	idColumnWidth int = 16
	// rule prefix field column width in table output
	prefixColumnWidth int = 16
	// StatusColumnWidth column width in table output
	statusColumnWidth int = 12
	// ExpiryColumnWidth column width in table output
	expiryColumnWidth int = 8
	// ExpiryDatesColumnWidth column width in table output
	expiryDatesColumnWidth int = 14
	// TagsColumnWidth column width in table output
	tagsColumnWidth int = 18
	// TransitionColumnWidth column width in table output
	transitionColumnWidth int = 14
	// TransitionDateColumnWidth column width in table output
	transitionDateColumnWidth int = 18
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
	expiryDatesLabel    string = "Date/Days "
	tagLabel            string = "Tags"
	transitionLabel     string = "Transition"
	transitionDateLabel string = "Date/Days "
	storageClassLabel   string = "Storage-Class "
)

// Keys to be used in map structure which stores the columns to be displayed.
const (
	statusLabelKey          string = "Enabled"
	storageClassLabelKey    string = "Storage-Class"
	expiryDatesLabelKey     string = "Expiry-Dates"
	transitionDatesLabelKey string = "Transition-Date"
)

// Some cell values
const (
	tickCell      string = "\u2713 "
	crossTickCell string = "\u2717 "
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
	minimum      bool
}

// PopulateILMDataForDisplay based on showDetails determined by user input, populate the ILM display
// table with information. Table is constructed row-by-row. Headers are first, then the rest of the rows.
func PopulateILMDataForDisplay(ilmCfg LifecycleConfiguration, rowCheck *map[string]int, alignedHdrLabels *[]string,
	cellDataNoTags *[][]string, cellDataWithTags *[][]string, tagRows *map[string][]string,
	showAll, showMin, showExpiry, showTransition bool) {

	// We need the different column headers and their respective column index
	// where they appear in a map data-structure format.
	// [Column Label] -> [Column Number]
	*rowCheck = make(map[string]int)
	// For rows with tags only tags are shown. Rest of the cells are empty (blanks in full cell length)
	*tagRows = make(map[string][]string)
	showOpts := showDetails{
		allAvailable: showAll,
		minimum:      showMin,
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
	colWidth[expiryLabel] = expiryColumnWidth
	colWidth[expiryDatesLabelKey] = expiryDatesColumnWidth
	colWidth[transitionLabel] = transitionColumnWidth
	colWidth[transitionDatesLabelKey] = transitionDateColumnWidth
	colWidth[storageClassLabelKey] = storageClassColumnWidth
	colWidth[tagLabel] = tagsColumnWidth

	return colWidth
}

// checkAddTableCellRows multiple rows created by filling up each cell of the table.
// Multiple rows are required for display of data with tags.
// Each 'key:value' pair is shown in 1 row and the rest of it is cells populated with blanks.
func checkAddTableCellRows(rowArr *[]string, rowCheck map[string]int, showOpts showDetails,
	cellInfo tableCellInfo, ruleID string, newRows map[string][]string) {
	var cellLabel string
	if showOpts.minimum {
		return
	}
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

// The right kind of tick is returned. Cross-tick if expiry is not set.
func getExpiryTick(rule LifecycleRule) string {
	expiryTick := crossTickCell
	expiryDateSet := rule.Expiration != nil && rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()
	expirySet := rule.Expiration != nil && (expiryDateSet || rule.Expiration.ExpirationInDays > 0)
	if expirySet {
		expiryTick = tickCell
	}
	return expiryTick
}

// The right kind of tick is returned. Cross-tick if status is 'Disabled' & tick if status is 'Enabled'.
func getStatusTick(rule LifecycleRule) string {
	statusTick := crossTickCell
	if rule.Status == statusLabelKey {
		statusTick = tickCell
	}
	return statusTick
}

// Expiry date. 'YYYY-MM-DD'. Set for 00:00:00 GMT as per the standard.
func getExpiryDateVal(rule LifecycleRule) string {
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

// Cross-tick if Transition is not set.
func getTransitionTick(rule LifecycleRule) string {
	transitionSet := rule.Transition != nil
	transitionDateSet := transitionSet && ((rule.Transition.TransitionDate != nil &&
		!rule.Transition.TransitionDate.IsZero()) ||
		rule.Transition.TransitionInDays > 0)
	if !transitionSet || !transitionDateSet {
		return crossTickCell
	}
	return tickCell
}

// Transition date. 'YYYY-MM-DD'. Set for 00:00:00 GMT as per the standard.
func getTransitionDate(rule LifecycleRule) string {
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

// Storage class name for transition.
func getStorageClassName(rule LifecycleRule) string {
	storageClass := blankCell
	transitionSet := (rule.Transition != nil)
	storageClassAvail := transitionSet && (rule.Transition.StorageClass != "")
	if storageClassAvail {
		storageClass = rule.Transition.StorageClass
	}
	return storageClass
}

// Array of Tag strings, each in key:value format
func getTagArr(rule LifecycleRule) []string {
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
func getILMShowDataWithoutTags(cellInfo *[][]string, rowCheck map[string]int, info LifecycleConfiguration, showOpts showDetails) {
	*cellInfo = make([][]string, 0)
	count := 0
	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]

		showExpiry := (rule.Expiration != nil) && ((rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()) ||
			rule.Expiration.ExpirationInDays > 0)
		transitionSet := (rule.Transition != nil) && ((rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero()) ||
			rule.Transition.TransitionInDays > 0)
		skipExpTran := (showOpts.expiry && !showExpiry) || (showOpts.transition && !transitionSet)
		if skipExpTran {
			continue
		}
		tagPresent := (rule.RuleFilter != nil) && (rule.RuleFilter.And != nil)
		if len(rule.TagFilters) > 0 || (tagPresent && len(rule.RuleFilter.And.Tags) > 0) {
			continue
		}
		*cellInfo = append(*cellInfo, make([]string, 0))
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: rule.ID, labelKey: idLabel, columnWidth: idColumnWidth, align: leftAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getPrefixVal(rule), labelKey: prefixLabel, columnWidth: prefixColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStatusTick(rule), labelKey: statusLabelKey, columnWidth: statusColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getExpiryTick(rule), labelKey: expiryLabel, columnWidth: expiryColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getExpiryDateVal(rule), labelKey: expiryDatesLabelKey, columnWidth: expiryDatesColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getTransitionTick(rule), labelKey: transitionLabel, columnWidth: transitionColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getTransitionDate(rule), labelKey: transitionDatesLabelKey, columnWidth: transitionDateColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStorageClassName(rule), labelKey: storageClassLabelKey, columnWidth: storageClassColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: blankCell, labelKey: tagLabel, columnWidth: tagsColumnWidth, align: centerAlign})
		count++
	}
}

// GetILMShowDataWithTags Just the data with extra rows for extra tags
func getILMShowDataWithTags(cellInfo *[][]string, newRows map[string][]string, rowCheck map[string]int, info LifecycleConfiguration, showOpts showDetails) {
	*cellInfo = make([][]string, 0)
	count := 0
	for index := 0; index < len(info.Rules); index++ {
		rule := info.Rules[index]

		showExpiry := (rule.Expiration != nil) && ((rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()) ||
			rule.Expiration.ExpirationInDays > 0)
		transitionSet := (rule.Transition != nil) && ((rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero()) ||
			rule.Transition.TransitionInDays > 0)
		skipExpTran := (showOpts.expiry && !showExpiry) || (showOpts.transition && !transitionSet)
		if skipExpTran {
			continue
		}
		if len(getTagArr(rule)) == 0 {
			continue
		}
		*cellInfo = append(*cellInfo, make([]string, 0))
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: rule.ID, labelKey: idLabel, columnWidth: idColumnWidth, align: leftAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getPrefixVal(rule), labelKey: prefixLabel, columnWidth: prefixColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStatusTick(rule), labelKey: statusLabelKey, columnWidth: statusColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getExpiryTick(rule), labelKey: expiryLabel, columnWidth: expiryColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getExpiryDateVal(rule), labelKey: expiryDatesLabelKey, columnWidth: expiryDatesColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getTransitionTick(rule), labelKey: transitionLabel, columnWidth: transitionColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getTransitionDate(rule), labelKey: transitionDatesLabelKey, columnWidth: transitionDateColumnWidth, align: centerAlign})
		checkAddTableCell(&((*cellInfo)[count]), rowCheck,
			tableCellInfo{label: getStorageClassName(rule), labelKey: storageClassLabelKey, columnWidth: storageClassColumnWidth, align: centerAlign})
		checkAddTableCellRows(&((*cellInfo)[count]), rowCheck, showOpts,
			tableCellInfo{multLabels: getTagArr(rule), label: "", labelKey: tagLabel, columnWidth: tagsColumnWidth, align: leftAlign},
			rule.ID, newRows)
		count++
	}
}

func getPrefixVal(rule LifecycleRule) string {
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

func showExpiryDetails(rule LifecycleRule, showOpts showDetails) bool {
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

func showTransitionDetails(rule LifecycleRule, showOpts showDetails) bool {
	if showOpts.allAvailable {
		return true
	}
	transitionSet := (rule.Transition != nil) &&
		((rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero()) ||
			rule.Transition.TransitionInDays > 0)
	transitionDetailsShow := (showOpts.allAvailable || showOpts.transition)
	return transitionSet && transitionDetailsShow
}

func showTags(rule LifecycleRule, showOpts showDetails) bool {
	if showOpts.minimum {
		return false
	}
	tagSet := showOpts.allAvailable ||
		(len(rule.TagFilters) > 0 || (rule.RuleFilter != nil && rule.RuleFilter.And != nil && (len(rule.RuleFilter.And.Tags) > 0)))
	return tagSet
}

func getColumns(info LifecycleConfiguration, rowCheck map[string]int, alignedHdrLabels *[]string, showOpts showDetails) {
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
		_, ok = rowCheck[expiryLabel]
		if !ok && showExpTick(showOpts) {
			rowCheck[expiryLabel] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(expiryLabel, centerAlign, colWidthTbl[expiryLabel]))
		}
		_, ok = rowCheck[expiryDatesLabelKey]
		if !ok && showExpiryDetails(rule, showOpts) {
			rowCheck[expiryDatesLabelKey] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(expiryDatesLabel, centerAlign, colWidthTbl[expiryDatesLabelKey]))
		}
		_, ok = rowCheck[transitionLabel]
		if !ok && showTransitionTick(showOpts) {
			rowCheck[transitionLabel] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(transitionLabel, centerAlign, colWidthTbl[transitionLabel]))
		}
		_, ok = rowCheck[transitionDatesLabelKey]
		if !ok && showTransitionDetails(rule, showOpts) {
			rowCheck[transitionDatesLabelKey] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(transitionDateLabel, centerAlign, colWidthTbl[transitionDatesLabelKey]))
		}
		_, ok = rowCheck[storageClassLabelKey]
		if !ok && showTransitionDetails(rule, showOpts) {
			rowCheck[storageClassLabelKey] = incColIdx()
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(storageClassLabel, centerAlign, colWidthTbl[storageClassLabelKey]))
		}
		_, ok = rowCheck[tagLabel]
		if !ok && showTags(rule, showOpts) {
			rowCheck[tagLabel] = incColIdx()
			tagIn = true
			(*alignedHdrLabels) = append((*alignedHdrLabels), getAlignedText(tagLabel, centerAlign, colWidthTbl[tagLabel]))
		}
	}
}
