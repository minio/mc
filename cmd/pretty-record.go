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
	"fmt"

	"github.com/minio/minio/pkg/console"
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
