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
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/minio/pkg/console"
)

// fixateScanBar truncates or stretches text to fit within the terminal size.
func fixateScanBar(text string, width int) string {
	if len([]rune(text)) > width {
		// Trim text to fit within the screen
		trimSize := len([]rune(text)) - width + 3 //"..."
		if trimSize < len([]rune(text)) {
			text = "..." + text[trimSize:]
		}
	} else {
		text += strings.Repeat(" ", width-len([]rune(text)))
	}
	return text
}

// Progress bar function report objects being scaned.
type scanBarFunc func(string)

// scanBarFactory returns a progress bar function to report URL scanning.
func scanBarFactory() scanBarFunc {
	fileCount := 0

	// Cursor animate channel.
	cursorCh := cursorAnimate()
	return func(source string) {
		scanPrefix := fmt.Sprintf("[%s] %s ", humanize.Comma(int64(fileCount)), <-cursorCh)
		source = fixateScanBar(source, globalTermWidth-len([]rune(scanPrefix)))
		barText := scanPrefix + source
		console.PrintC("\r" + barText + "\r")
		fileCount++
	}
}
