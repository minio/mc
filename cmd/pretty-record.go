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

// Row specifies row description and theme
type Row struct {
	desc      string
	descTheme string
}

// PrettyRecord - an easy struct to format a set of key-value
// pairs into a record
type PrettyRecord struct {
	rows   []Row
	indent int
	maxLen int
}

// newPrettyRecord - creates a new pretty record
func newPrettyRecord(indent int, rows ...Row) PrettyRecord {
	maxDescLen := 0
	for _, r := range rows {
		if len(r.desc) > maxDescLen {
			maxDescLen = len(r.desc)
		}
	}
	return PrettyRecord{
		rows:   rows,
		indent: indent,
		maxLen: maxDescLen,
	}
}

// buildRecord - creates a string which represents a record table given
// some fields contents.
func (t PrettyRecord) buildRecord(contents ...string) (line string) {

	// totalRows is the minimum of the number of fields config
	// and the number of contents elements.
	totalRows := len(contents)
	if len(t.rows) < totalRows {
		totalRows = len(t.rows)
	}
	var format, separator string
	// Format fields and construct message
	for i := 0; i < totalRows; i++ {
		// default heading
		indent := 0
		format = "%s\n"
		// optionally indented rows with key value pairs
		if i > 0 {
			indent = t.indent
			format = fmt.Sprintf("%%%ds%%-%ds : %%s\n", indent, t.maxLen)
			line += console.Colorize(t.rows[i].descTheme, fmt.Sprintf(format, separator, t.rows[i].desc, contents[i]))
		} else {
			line += console.Colorize(t.rows[i].descTheme, fmt.Sprintf(format, contents[i]))
		}
	}
	return
}
