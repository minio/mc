/*
 * Mini Object Storage, (C) 2015 Minio, Inc.
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
	"os"

	"github.com/mgutz/ansi"
)

// Color coding format "foregroundColor+attributes:backgroundColor+attributes"
func fatal(msg string) {
	red := ansi.ColorFunc("red+B")
	fmt.Println(red(msg))
	os.Exit(1)
}

func warning(msg string) {
	yellow := ansi.ColorFunc("yellow")
	fmt.Println(yellow(msg))
}

func info(msg string) {
	green := ansi.ColorFunc("green")
	fmt.Println(green(msg))
}

func infoCallback(msg string) {
	green := ansi.ColorFunc("green")
	fmt.Print("\r" + green(msg))
}
