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

package console

import (
	"os"

	"github.com/fatih/color"
)

// Fatal - print in red and exit
func Fatal(msgs ...interface{}) {
	red := color.New(color.FgRed, color.Bold)
	colorPrint(red, msgs)
	os.Exit(1)
}

// Fatalln - print in red with newline and exit
func Fatalln(msgs ...interface{}) {
	red := color.New(color.FgRed, color.Bold)
	colorPrintln(red, msgs)
	os.Exit(1)
}

// Fatalf - format in red and exit
func Fatalf(format string, msgs ...interface{}) {
	red := color.New(color.FgRed, color.Bold)
	red.Printf(format, msgs)
	os.Exit(1)
}

// Error - print in yellow
func Error(msgs ...interface{}) {
	yellow := color.New(color.FgYellow, color.Bold)
	colorPrint(yellow, msgs)
}

// Errorln - print in yellow with newline
func Errorln(msgs ...interface{}) {
	yellow := color.New(color.FgYellow, color.Bold)
	colorPrintln(yellow, msgs)
}

// Info - print in green
func Info(msgs ...interface{}) {
	green := color.New(color.FgGreen, color.Bold)
	colorPrint(green, msgs)
}

// Infoln - print in green with newline
func Infoln(msgs ...interface{}) {
	green := color.New(color.FgGreen, color.Bold)
	colorPrintln(green, msgs)
}

// Infof - format in green
func Infof(format string, msgs ...interface{}) {
	green := color.New(color.FgGreen, color.Bold)
	green.Printf(format, msgs)
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
