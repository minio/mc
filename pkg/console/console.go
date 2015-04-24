/*
 * Mini Copy (C) 2015 Minio, Inc.
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
	"runtime"
	"sync"

	"path/filepath"

	"github.com/fatih/color"
	"github.com/minio-io/minio/pkg/iodine"
)

var (
	mutex = &sync.RWMutex{}

	// themesDB contains supported list of Themes
	themesDB = map[string]Theme{"minimal": MiniTheme, "nocolor": NoColorTheme, "white": WhiteTheme}

	// currTheme is current theme
	currThemeName = GetDefaultThemeName()

	// Fatal prints a error message and exits
	Fatal = func(a ...interface{}) { print(themesDB[currThemeName].Fatal, "<FATAL>", a...); os.Exit(1) }
	// Fatalln prints a error message with a new line and exits
	Fatalln = func(a ...interface{}) { println(themesDB[currThemeName].Fatal, "<FATAL>", a...); os.Exit(1) }
	// Fatalf prints a error message with formatting and exits
	Fatalf = func(f string, a ...interface{}) {
		printf(themesDB[currThemeName].Fatal, "<FATAL>", f, a...)
		os.Exit(1)
	}

	// Error prints a error message
	Error = func(a ...interface{}) { print(themesDB[currThemeName].Error, "<ERROR>", a...) }
	// Errorln prints a error message with a new line
	Errorln = func(a ...interface{}) { println(themesDB[currThemeName].Error, "<ERROR>", a...) }
	// Errorf prints a error message with formatting
	Errorf = func(f string, a ...interface{}) { printf(themesDB[currThemeName].Error, "<ERROR>", f, a...) }

	// Info prints a informational message
	Info = func(a ...interface{}) { print(themesDB[currThemeName].Info, "", a...) }
	// Infoln prints a informational message with a new line
	Infoln = func(a ...interface{}) { println(themesDB[currThemeName].Info, "", a...) }
	// Infof prints a informational message with formatting
	Infof = func(f string, a ...interface{}) { printf(themesDB[currThemeName].Info, "", f, a...) }

	// Debug prints a debug message
	Debug = func(a ...interface{}) { print(themesDB[currThemeName].Debug, "<DEBUG>", a...) }
	// Debugln prints a debug message with a new line
	Debugln = func(a ...interface{}) { println(themesDB[currThemeName].Debug, "<DEBUG>", a...) }
	// Debugf prints a debug message with formatting
	Debugf = func(f string, a ...interface{}) { printf(themesDB[currThemeName].Debug, "<DEBUG>", f, a...) }

	// File - File("foo.txt")
	File = themesDB[currThemeName].File.SprintfFunc()
	// Dir - Dir("dir/")
	Dir = themesDB[currThemeName].Dir.SprintfFunc()
	// Size - Size("12GB")
	Size = themesDB[currThemeName].Size.SprintfFunc()
	//Time - Time("12 Hours")
	Time = themesDB[currThemeName].Time.SprintfFunc()
)

// Theme holds console color scheme
type Theme struct {
	Fatal *color.Color
	Error *color.Color
	Info  *color.Color
	Debug *color.Color
	Size  *color.Color
	Time  *color.Color
	File  *color.Color
	Dir   *color.Color
	//	Reason *color.Color
}

var (
	// wrap around standard fmt functions
	// print prints a message prefixed with message type and program name
	print = func(c *color.Color, prefix string, a ...interface{}) {
		mutex.Lock()
		c.Printf(ProgramName()+": %s ", prefix)
		c.Print(a...)
		mutex.Unlock()
	}

	// println - same as print with a new line
	println = func(c *color.Color, prefix string, a ...interface{}) {
		mutex.Lock()
		c.Printf(ProgramName()+": %s ", prefix)
		c.Println(a...)
		mutex.Unlock()
	}

	// printf - same as print, but takes a format specifier
	printf = func(c *color.Color, prefix string, f string, a ...interface{}) {
		mutex.Lock()
		c.Printf(ProgramName()+": %s ", prefix)
		c.Printf(f, a...)
		mutex.Unlock()
	}
)

// SetTheme sets a color theme
func SetTheme(themeName string) error {
	if !IsValidTheme(themeName) {
		return iodine.New(fmt.Errorf("Unsupported theme name [%s]", themeName), nil)
	}

	mutex.Lock()

	currThemeName = themeName
	theme := themesDB[currThemeName]

	// Just another additional precaution to completely disable color.
	// Color theme is also necessary, because it does other useful things like exit-on-fatal..
	if currThemeName == "nocolor" {
		color.NoColor = true
	} else {
		color.NoColor = false
	}

	// Error prints a error message
	Error = func(a ...interface{}) { print(theme.Error, "<ERROR>", a...) }
	// Errorln prints a error message with a new line
	Errorln = func(a ...interface{}) { println(theme.Error, "<ERROR>", a...) }
	// Errorf prints a error message with formatting
	Errorf = func(f string, a ...interface{}) { printf(theme.Error, "<ERROR>", f, a...) }

	// Info prints a informational message
	Info = func(a ...interface{}) { print(theme.Info, "", a...) }
	// Infoln prints a informational message with a new line
	Infoln = func(a ...interface{}) { println(theme.Info, "", a...) }
	// Infof prints a informational message with formatting
	Infof = func(f string, a ...interface{}) { printf(theme.Info, "", f, a...) }

	// Debug prints a debug message
	Debug = func(a ...interface{}) { print(theme.Debug, "<DEBUG>", a...) }
	// Debugln prints a debug message with a new line
	Debugln = func(a ...interface{}) { println(theme.Debug, "<DEBUG>", a...) }
	// Debugf prints a debug message with formatting
	Debugf = func(f string, a ...interface{}) { printf(theme.Debug, "<DEBUG>", f, a...) }

	Dir = theme.Dir.SprintfFunc()
	File = theme.File.SprintfFunc()
	Size = theme.Size.SprintfFunc()
	Time = theme.Time.SprintfFunc()

	mutex.Unlock()

	return nil
}

// GetThemeName returns currently set theme name
func GetThemeName() string {
	return currThemeName
}

// GetDefaultThemeName returns the default theme
func GetDefaultThemeName() string {
	if runtime.GOOS == "windows" {
		return "nocolor"
	}
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

// ProgramName - return the name of the executable program
func ProgramName() string {
	_, progName := filepath.Split(os.Args[0])
	return progName
}
