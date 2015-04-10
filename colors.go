/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"os"

	"github.com/fatih/color"
)

func fatal(msgs ...interface{}) {
	red := color.New(color.FgRed)
	boldRed := red.Add(color.Bold)
	colorPrintln(boldRed, msgs)
	os.Exit(1)
}

func info(msgs ...interface{}) {
	if !globalQuietFlag {
		green := color.New(color.FgGreen)
		boldGreen := green.Add(color.Bold)
		colorPrintln(boldGreen, msgs)
	}
}

func infoCallback(msg string) {
	if !globalQuietFlag {
		green := color.New(color.FgGreen)
		boldGreen := green.Add(color.Bold)
		boldGreen.Print("\r" + msg)
	}
}

func colorPrint(c *color.Color, msgs []interface{}) {
	for i, msg := range msgs {
		if i != 0 {
			c.Print(" ")
		}
		c.Print(msg)
	}
}

func colorPrintln(c *color.Color, msgs []interface{}) {
	colorPrint(c, msgs)
	c.Println()
}
