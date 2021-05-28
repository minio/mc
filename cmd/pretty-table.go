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

	"github.com/minio/pkg/console"
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
