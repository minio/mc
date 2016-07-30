/*
 * Minio Client (C) 2014-2016 Minio, Inc.
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

package command

import (
	"fmt"
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
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

func discardScanBarFactory(s string) {
}

// scanBarFactory returns a progress bar function to report URL scanning.
func scanBarFactory() scanBarFunc {
	fileCount := 0
	termWidth, e := pb.GetTerminalWidth()
	fatalIf(probe.NewError(e), "Unable to get terminal size. Please use --quiet option.")

	// Cursor animate channel.
	cursorCh := cursorAnimate()
	return func(source string) {
		scanPrefix := fmt.Sprintf("[%s] %s ", humanize.Comma(int64(fileCount)), string(<-cursorCh))
		source = fixateScanBar(source, termWidth-len([]rune(scanPrefix)))
		barText := scanPrefix + source
		console.PrintC("\r" + barText + "\r")
		fileCount++
	}
}
