/*
 * MinIO Client (C) 2018 MinIO, Inc.
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

import "github.com/fatih/color"

// An alias of string to represent the health color code of an object
type col string

const (
	colGrey   col = "Grey"
	colRed    col = "Red"
	colYellow col = "Yellow"
	colGreen  col = "Green"
)

// getPrintCol - map color code to color for printing
func getPrintCol(c col) *color.Color {
	switch c {
	case colGrey:
		return color.New(color.FgWhite, color.Bold)
	case colRed:
		return color.New(color.FgRed, color.Bold)
	case colYellow:
		return color.New(color.FgYellow, color.Bold)
	case colGreen:
		return color.New(color.FgGreen, color.Bold)
	}
	return nil
}
