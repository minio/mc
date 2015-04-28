/*
 * Minio Client (C) 2015 Minio, Inc.
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

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

// go get update string
const (
	updateString = "go get -u github.com/minio-io/mc"
)

// intMax - return maximum value for any given integer
func intMax(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// printUpdateNotify - inspired from Yeoman project npm package https://github.com/yeoman/update-notifier
func printUpdateNotify(latestVersion, currentVersion string) {
	// TODO - make this configurable
	//
	// initialize coloring
	green := color.New(color.FgGreen)
	boldGreen := green.Add(color.Bold).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	blueFaint := color.New(color.FgBlue, color.BlinkSlow).SprintfFunc()
	yellow := color.New(color.FgYellow).SprintfFunc()

	// calculate length without color coding, due to ANSI color characters padded to actual
	// string the final length is wrong than the original string length
	line1Length := len(fmt.Sprintf("  Update available: %s (current: %s)", latestVersion, currentVersion))
	line2Length := len(fmt.Sprintf("  Run %s to update. ", updateString))

	// populate lines with color coding
	line1InColor := fmt.Sprintf("  Update available: %s (current: %s)", boldGreen(latestVersion), blueFaint(currentVersion))
	line2InColor := fmt.Sprintf("  Run %s to update. ", blue(updateString))

	// calculate the rectangular box size
	maxContentWidth := intMax(line1Length, line2Length)
	line1Rest := maxContentWidth - line1Length
	line2Rest := maxContentWidth - line2Length

	// on windows terminal turn off unicode characters
	var top, bottom, sideBar string
	if runtime.GOOS == "windows" {
		top = yellow("*" + strings.Repeat("*", maxContentWidth) + "*")
		bottom = yellow("*" + strings.Repeat("*", maxContentWidth) + "*")
		sideBar = yellow("|")
	} else {
		// color the rectangular box, use unicode characters here
		top = yellow("┏" + strings.Repeat("━", maxContentWidth) + "┓")
		bottom = yellow("┗" + strings.Repeat("━", maxContentWidth) + "┛")
		sideBar = yellow("┃")
	}
	// fill spaces to the rest of the area
	spacePaddingLine1 := strings.Repeat(" ", line1Rest)
	spacePaddingLine2 := strings.Repeat(" ", line2Rest)

	// construct the final message
	message := "\n" + top + "\n" +
		sideBar + line1InColor + spacePaddingLine1 + sideBar + "\n" +
		sideBar + line2InColor + spacePaddingLine2 + sideBar + "\n" +
		bottom + "\n"

	// finally print the message
	fmt.Println(message)
}
