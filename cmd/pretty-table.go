/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
	"fmt"

	"github.com/minio/minio/pkg/console"
)

// Field configuration: color theme and max content length
type Field struct {
	colorTheme string
	maxLen     int
}

// PrettyTable - an easy struct to format a set of line
type PrettyTable struct {
	cols      []Field
	separator string
}

// newPrettyTable - creates a new pretty table
func newPrettyTable(separator string, cols ...Field) PrettyTable {
	return PrettyTable{
		cols:      cols,
		separator: separator,
	}
}

// buildRow - creates a string which represents a line table given
// some fields contents.
func (t PrettyTable) buildRow(contents ...string) (line string) {
	dots := "..."

	// totalColumns is the minimum of the number of fields config
	// and the number of contents elements.
	totalColumns := len(contents)
	if len(t.cols) < totalColumns {
		totalColumns = len(t.cols)
	}

	// Format fields and construct message
	for i := 0; i < totalColumns; i++ {
		// Default field format without pretty effect
		fieldContent := ""
		fieldFormat := "%s"
		if t.cols[i].maxLen >= 0 {
			// Override field format
			fieldFormat = fmt.Sprintf("%%-%d.%ds", t.cols[i].maxLen, t.cols[i].maxLen)
			// Cut field string and add '...' if length is greater than maxLen
			if len(contents[i]) > t.cols[i].maxLen {
				fieldContent = contents[i][:t.cols[i].maxLen-len(dots)] + dots
			} else {
				fieldContent = contents[i]
			}
		} else {
			fieldContent = contents[i]
		}

		// Add separator if this is not the last column
		if i < totalColumns-1 {
			fieldFormat += t.separator
		}

		// Add the field to the resulted message
		line += console.Colorize(t.cols[i].colorTheme, fmt.Sprintf(fieldFormat, fieldContent))
	}
	return
}
