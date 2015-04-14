/*
 * Modern Copy, (C) 2015 Minio, Inc.
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
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
)

var (
	mutex     = &sync.RWMutex{}
	currTheme = "minimal"

	// ThemesDB contains supported list of Themes
	ThemesDB = map[string]Theme{"minimal": MiniTheme, "nocolor": NoColorTheme, "white": WhiteTheme}
	// Fatal prints a fatal message and exits
	Fatal = ThemesDB[currTheme].Fatal.PrintFunc()
	// Fatalln prints a fatal message with a new line and exits
	Fatalln = ThemesDB[currTheme].Fatal.PrintlnFunc()
	// Fatalf prints a fatal message with formatting and exits
	Fatalf = ThemesDB[currTheme].Fatal.PrintfFunc()
	// Error prints a error message
	Error = ThemesDB[currTheme].Error.PrintFunc()
	// Errorln prints a error message with a new line
	Errorln = ThemesDB[currTheme].Error.PrintlnFunc()
	// Errorf prints a error message with formatting
	Errorf = ThemesDB[currTheme].Error.PrintfFunc()
	// Info prints a informational message
	Info = ThemesDB[currTheme].Info.PrintFunc()
	// Infoln prints a informational message with a new line
	Infoln = ThemesDB[currTheme].Info.PrintlnFunc()
	// Infof prints a informational message with formatting
	Infof = ThemesDB[currTheme].Info.PrintfFunc()
	// Debug prints a debug message
	Debug = ThemesDB[currTheme].Debug.PrintFunc()
	// Debugln prints a debug message with a new line
	Debugln = ThemesDB[currTheme].Debug.PrintlnFunc()
	// Debugf prints a debug message with formatting
	Debugf = ThemesDB[currTheme].Debug.PrintfFunc()
)

// Theme holds console color scheme
type Theme struct {
	Fatal *color.Color
	Error *color.Color
	Info  *color.Color
	Debug *color.Color
}

// MiniTheme is a color scheme set by minimal
var MiniTheme = Theme{
	Fatal: (color.New(color.FgRed, color.Bold)),
	Error: (color.New(color.FgYellow, color.Bold)),
	Info:  (color.New(color.FgGreen, color.Bold)),
	Debug: (color.New(color.FgCyan, color.Bold)),
}

// MiniTheme is a color scheme set by minimal
var WhiteTheme = Theme{
	Fatal: (color.New(color.FgWhite, color.Bold)),
	Error: (color.New(color.FgWhite, color.Bold)),
	Info:  (color.New(color.FgWhite, color.Bold)),
	Debug: (color.New(color.FgWhite, color.Bold)),
}

// NoColorTheme disables color theme
var NoColorTheme = Theme{}

func isValidTheme(name string) bool {
	for key := range ThemesDB {
		if key == name {
			return true
		}
	}
	return false
}

// SetTheme sets a color theme
func SetTheme(name string) error {
	mutex.Lock()
	currTheme = name
	if currTheme == "" {
		currTheme = "minimal"
	}
	if !isValidTheme(currTheme) {
		return errors.New("Invalid theme")
	}

	print := func(a ...interface{}) { fmt.Print(a...) }
	println := func(a ...interface{}) { fmt.Println(a...) }
	printf := func(f string, a ...interface{}) { fmt.Printf(f, a...) }

	fatalPrint := func(a ...interface{}) { fmt.Print(a...); os.Exit(1) }
	fatalPrintln := func(a ...interface{}) { fmt.Println(a...); os.Exit(1) }
	fatalPrintf := func(f string, a ...interface{}) { fmt.Printf(f, a...); os.Exit(1) }

	if currTheme == "nocolor" {

		Fatal = fatalPrint
		Fatalln = fatalPrintln
		Fatalf = fatalPrintf
		Error = print
		Errorln = println
		Errorf = printf
		Info = print
		Infoln = println
		Infof = printf
		Debug = print
		Debugln = println
		Debugf = printf
	} else {
		if currTheme == "" {
			currTheme = "minimal"
		}
		Fatal = func(a ...interface{}) { ThemesDB[currTheme].Fatal.Print(a...); os.Exit(1) }
		Fatalln = func(a ...interface{}) { ThemesDB[currTheme].Fatal.Println(a...); os.Exit(1) }
		Fatalf = func(f string, a ...interface{}) { ThemesDB[currTheme].Fatal.Printf(f, a...); os.Exit(1) }
		Error = func(a ...interface{}) { ThemesDB[currTheme].Error.Print(a...) }
		Errorln = func(a ...interface{}) { ThemesDB[currTheme].Error.Println(a...) }
		Errorf = func(f string, a ...interface{}) { ThemesDB[currTheme].Error.Printf(f, a...) }
		Info = func(a ...interface{}) { ThemesDB[currTheme].Info.Print(a...) }
		Infoln = func(a ...interface{}) { ThemesDB[currTheme].Info.Println(a...) }
		Infof = func(f string, a ...interface{}) { ThemesDB[currTheme].Info.Printf(f, a...) }
		Debug = func(a ...interface{}) { ThemesDB[currTheme].Debug.Print(a...) }
		Debugln = func(a ...interface{}) { ThemesDB[currTheme].Debug.Println(a...) }
		Debugf = func(f string, a ...interface{}) { ThemesDB[currTheme].Debug.Printf(f, a...) }
	}
	mutex.Unlock()
	return nil
}

// GetTheme returns currently set theme
func GetTheme() string {
	return currTheme
}
