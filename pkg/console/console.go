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
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/minio-io/minio/pkg/iodine"
)

var (
	mutex = &sync.RWMutex{}

	// themesDB contains supported list of Themes
	themesDB = map[string]Theme{"minimal": MiniTheme, "nocolor": NoColorTheme, "white": WhiteTheme}

	// currTheme is current theme
	currThemeName = GetDefaultThemeName()

	// Fatal prints a fatal message and exits
	Fatal = func(a ...interface{}) { themesDB[currThemeName].Fatal.Print(a...); os.Exit(1) }
	// Fatalln prints a fatal message with a new line and exits
	Fatalln = func(a ...interface{}) { themesDB[currThemeName].Fatal.Println(a...); os.Exit(1) }
	// Fatalf prints a fatal message with formatting and exits
	Fatalf = func(f string, a ...interface{}) { themesDB[currThemeName].Fatal.Printf(f, a...); os.Exit(1) }
	// Error prints a error message
	Error = func(a ...interface{}) { themesDB[currThemeName].Error.Print(a...) }
	// Errorln prints a error message with a new line
	Errorln = func(a ...interface{}) { themesDB[currThemeName].Error.Println(a...) }
	// Errorf prints a error message with formatting
	Errorf = func(f string, a ...interface{}) { themesDB[currThemeName].Error.Printf(f, a...) }
	// Info prints a informational message
	Info = func(a ...interface{}) { themesDB[currThemeName].Info.Print(a...) }
	// Infoln prints a informational message with a new line
	Infoln = func(a ...interface{}) { themesDB[currThemeName].Info.Println(a...) }
	// Infof prints a informational message with formatting
	Infof = func(f string, a ...interface{}) { themesDB[currThemeName].Info.Printf(f, a...) }
	// Debug prints a debug message
	Debug = func(a ...interface{}) { themesDB[currThemeName].Debug.Print(a...) }
	// Debugln prints a debug message with a new line
	Debugln = func(a ...interface{}) { themesDB[currThemeName].Debug.Println(a...) }
	// Debugf prints a debug message with formatting
	Debugf = func(f string, a ...interface{}) { themesDB[currThemeName].Debug.Printf(f, a...) }
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
var NoColorTheme = Theme{
	Fatal: (color.New()),
	Error: (color.New()),
	Info:  (color.New()),
	Debug: (color.New()),
}

var (
	// wrap around standard fmt functions
	print   = func(a ...interface{}) { fmt.Print(a...) }
	println = func(a ...interface{}) { fmt.Println(a...) }
	printf  = func(f string, a ...interface{}) { fmt.Printf(f, a...) }

	fatalPrint   = func(a ...interface{}) { fmt.Print(a...); os.Exit(1) }
	fatalPrintln = func(a ...interface{}) { fmt.Println(a...); os.Exit(1) }
	fatalPrintf  = func(f string, a ...interface{}) { fmt.Printf(f, a...); os.Exit(1) }
)

// SetTheme sets a color theme
func SetTheme(themeName string) error {
	if !IsValidTheme(themeName) {
		return iodine.New(fmt.Errorf("Unsupported theme name [%s]", themeName), nil)
	}

	mutex.Lock()
	currThemeName = themeName
	theme := themesDB[currThemeName]

	Fatal = func(a ...interface{}) { theme.Fatal.Print(a...); os.Exit(1) }
	Fatalln = func(a ...interface{}) { theme.Fatal.Println(a...); os.Exit(1) }
	Fatalf = func(f string, a ...interface{}) { theme.Fatal.Printf(f, a...); os.Exit(1) }
	Error = func(a ...interface{}) { theme.Error.Print(a...) }
	Errorln = func(a ...interface{}) { theme.Error.Println(a...) }
	Errorf = func(f string, a ...interface{}) { theme.Error.Printf(f, a...) }
	Info = func(a ...interface{}) { theme.Info.Print(a...) }
	Infoln = func(a ...interface{}) { theme.Info.Println(a...) }
	Infof = func(f string, a ...interface{}) { theme.Info.Printf(f, a...) }
	Debug = func(a ...interface{}) { theme.Debug.Print(a...) }
	Debugln = func(a ...interface{}) { theme.Debug.Println(a...) }
	Debugf = func(f string, a ...interface{}) { theme.Debug.Printf(f, a...) }
	mutex.Unlock()

	return nil
}

// GetThemeName returns currently set theme name
func GetThemeName() string {
	return currThemeName
}

// GetDefaultThemeName returns the default theme
func GetDefaultThemeName() string {
	return "minimal"
}

// GetThemeNames returns currently supported list of  themes
func GetThemeNames() (themeNames []string) {

	for themeName := range themesDB {
		themeNames = append(themeNames, themeName)
	}
	return themeNames
}

// IsValidTheme returns true if "themeName" is currently supported
func IsValidTheme(themeName string) bool {
	_, ok := themesDB[themeName]
	return ok
}
