/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"runtime"
	"sync"

	"github.com/fatih/color"
	"github.com/minio-io/minio/pkg/iodine"
)

var (
	mutex = &sync.RWMutex{}

	// currentTheme is currently set theme
	currentTheme = GetDefaultTheme()
	// ThemesDB contains supported list of Themes
	ThemesDB = map[string]Theme{"minimal": MiniTheme, "nocolor": NoColorTheme, "white": WhiteTheme}
	// Fatal prints a fatal message and exits
	Fatal = ThemesDB[currentTheme].Fatal.PrintFunc()
	// Fatalln prints a fatal message with a new line and exits
	Fatalln = ThemesDB[currentTheme].Fatal.PrintlnFunc()
	// Fatalf prints a fatal message with formatting and exits
	Fatalf = ThemesDB[currentTheme].Fatal.PrintfFunc()
	// Error prints a error message
	Error = ThemesDB[currentTheme].Error.PrintFunc()
	// Errorln prints a error message with a new line
	Errorln = ThemesDB[currentTheme].Error.PrintlnFunc()
	// Errorf prints a error message with formatting
	Errorf = ThemesDB[currentTheme].Error.PrintfFunc()
	// Info prints a informational message
	Info = ThemesDB[currentTheme].Info.PrintFunc()
	// Infoln prints a informational message with a new line
	Infoln = ThemesDB[currentTheme].Info.PrintlnFunc()
	// Infof prints a informational message with formatting
	Infof = ThemesDB[currentTheme].Info.PrintfFunc()
	// Debug prints a debug message
	Debug = ThemesDB[currentTheme].Debug.PrintFunc()
	// Debugln prints a debug message with a new line
	Debugln = ThemesDB[currentTheme].Debug.PrintlnFunc()
	// Debugf prints a debug message with formatting
	Debugf = ThemesDB[currentTheme].Debug.PrintfFunc()
)

// Theme holds console color scheme
type Theme struct {
	Fatal *color.Color
	Error *color.Color
	Info  *color.Color
	Debug *color.Color
}

// MiniTheme is a color scheme with
//  - Red color for Fatal
//  - Yellow for Error
//  - Green for Info
//  - Cyan for Debug
var MiniTheme = Theme{
	Fatal: (color.New(color.FgRed, color.Bold)),
	Error: (color.New(color.FgYellow, color.Bold)),
	Info:  (color.New(color.FgGreen, color.Bold)),
	Debug: (color.New(color.FgCyan, color.Bold)),
}

// WhiteTheme is a color scheme with white colors only
var WhiteTheme = Theme{
	Fatal: (color.New(color.FgWhite, color.Bold)),
	Error: (color.New(color.FgWhite, color.Bold)),
	Info:  (color.New(color.FgWhite, color.Bold)),
	Debug: (color.New(color.FgWhite, color.Bold)),
}

// NoColorTheme disables color theme
var NoColorTheme = Theme{}

var (
	// wrap around standard fmt functions
	print   = func(a ...interface{}) { fmt.Print(a...) }
	println = func(a ...interface{}) { fmt.Println(a...) }
	printf  = func(f string, a ...interface{}) { fmt.Printf(f, a...) }

	fatalPrint   = func(a ...interface{}) { fmt.Print(a...); os.Exit(1) }
	fatalPrintln = func(a ...interface{}) { fmt.Println(a...); os.Exit(1) }
	fatalPrintf  = func(f string, a ...interface{}) { fmt.Printf(f, a...); os.Exit(1) }
)

// setThemeNoColor - set theme no color
func setThemeNoColor(themeName string) {
	mutex.Lock()
	currentTheme = themeName
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
	mutex.Unlock()
}

// setThemeColor - set theme color style
func setThemeColor(themeName string) {
	mutex.Lock()
	currentTheme = themeName
	Fatal = func(a ...interface{}) { ThemesDB[currentTheme].Fatal.Print(a...); os.Exit(1) }
	Fatalln = func(a ...interface{}) { ThemesDB[currentTheme].Fatal.Println(a...); os.Exit(1) }
	Fatalf = func(f string, a ...interface{}) { ThemesDB[currentTheme].Fatal.Printf(f, a...); os.Exit(1) }
	Error = func(a ...interface{}) { ThemesDB[currentTheme].Error.Print(a...) }
	Errorln = func(a ...interface{}) { ThemesDB[currentTheme].Error.Println(a...) }
	Errorf = func(f string, a ...interface{}) { ThemesDB[currentTheme].Error.Printf(f, a...) }
	Info = func(a ...interface{}) { ThemesDB[currentTheme].Info.Print(a...) }
	Infoln = func(a ...interface{}) { ThemesDB[currentTheme].Info.Println(a...) }
	Infof = func(f string, a ...interface{}) { ThemesDB[currentTheme].Info.Printf(f, a...) }
	Debug = func(a ...interface{}) { ThemesDB[currentTheme].Debug.Print(a...) }
	Debugln = func(a ...interface{}) { ThemesDB[currentTheme].Debug.Println(a...) }
	Debugf = func(f string, a ...interface{}) { ThemesDB[currentTheme].Debug.Printf(f, a...) }
	mutex.Unlock()
}

// SetTheme sets a color theme
func SetTheme(themeName string) error {
	switch true {
	case themeName == "nocolor":
		setThemeNoColor(themeName)
	case themeName == "minimal" || themeName == "white":
		setThemeColor(themeName)
	default:
		msg := fmt.Sprintf("Invalid theme: %s", themeName)
		return iodine.New(errors.New(msg), nil)
	}
	return nil
}

// GetTheme returns currently set theme
func GetTheme() string {
	return currentTheme
}

// GetDefaultTheme returns the default theme
func GetDefaultTheme() string {
	if runtime.GOOS == "windows" {
		return "nocolor"
	}
	return "minimal"
}
