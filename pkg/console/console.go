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

// NoDebugPrint defines if the input should be printed or not. By default it's set to true.
var NoDebugPrint = true

var (
	mutex = &sync.RWMutex{}

	// themesDB contains supported list of Themes
	themesDB = map[string]Theme{"minimal": MiniTheme, "nocolor": NoColorTheme, "white": WhiteTheme}

	// currTheme is current theme
	currThemeName = GetDefaultThemeName()

	// Print prints a error message and exits
	Print = themesDB[currThemeName].Info.Print
	// Println prints a error message with a new line and exits
	Println = themesDB[currThemeName].Info.Println
	// Printf prints a error message with formatting and exits
	Printf = themesDB[currThemeName].Info.Printf

	// Fatal prints a error message and exits
	Fatal = func(a ...interface{}) { print(themesDB[currThemeName].Fatal, a...); os.Exit(1) }
	// Fatalln prints a error message with a new line and exits
	Fatalln = func(a ...interface{}) { println(themesDB[currThemeName].Fatal, a...); os.Exit(1) }
	// Fatalf prints a error message with formatting and exits
	Fatalf = func(f string, a ...interface{}) { printf(themesDB[currThemeName].Fatal, f, a...); os.Exit(1) }

	// Error prints a error message
	Error = func(a ...interface{}) { print(themesDB[currThemeName].Error, a...) }
	// Errorln prints a error message with a new line
	Errorln = func(a ...interface{}) { println(themesDB[currThemeName].Error, a...) }
	// Errorf prints a error message with formatting
	Errorf = func(f string, a ...interface{}) { printf(themesDB[currThemeName].Error, f, a...) }

	// Info prints a informational message
	Info = func(a ...interface{}) { print(themesDB[currThemeName].Info, a...) }
	// Infoln prints a informational message with a new line
	Infoln = func(a ...interface{}) { println(themesDB[currThemeName].Info, a...) }
	// Infof prints a informational message with formatting
	Infof = func(f string, a ...interface{}) { printf(themesDB[currThemeName].Info, f, a...) }

	// Debug prints a debug message
	Debug = func(a ...interface{}) {
		if !NoDebugPrint {
			print(themesDB[currThemeName].Debug, a...)
		}
	}
	// Debugln prints a debug message with a new line
	Debugln = func(a ...interface{}) {
		if !NoDebugPrint {
			println(themesDB[currThemeName].Debug, a...)
		}
	}
	// Debugf prints a debug message with formatting
	Debugf = func(f string, a ...interface{}) {
		if !NoDebugPrint {
			printf(themesDB[currThemeName].Debug, f, a...)
		}
	}

	// File - File("foo.txt")
	File = themesDB[currThemeName].File.SprintfFunc()
	// Dir - Dir("dir/")
	Dir = themesDB[currThemeName].Dir.SprintfFunc()
	// Size - Size("12GB")
	Size = themesDB[currThemeName].Size.SprintfFunc()
	// Time - Time("12 Hours")
	Time = themesDB[currThemeName].Time.SprintfFunc()
	// Retry - Retry message number
	Retry = themesDB[currThemeName].Retry.SprintfFunc()
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
	Retry *color.Color
	//	Reason *color.Color
}

var (
	// wrap around standard fmt functions
	// print prints a message prefixed with message type and program name
	print = func(c *color.Color, a ...interface{}) {
		switch c {
		case themesDB[currThemeName].Debug:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Print(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Fatal:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <FATAL> ")
			c.Print(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Error:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <ERROR> ")
			c.Print(a...)
			color.Output = output
			mutex.Unlock()
		default:
			mutex.Lock()
			c.Print(ProgramName() + ": ")
			c.Print(a...)
			mutex.Unlock()
		}
	}

	// println - same as print with a new line
	println = func(c *color.Color, a ...interface{}) {
		switch c {
		case themesDB[currThemeName].Debug:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Println(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Fatal:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <FATAL> ")
			c.Println(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Error:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <ERROR> ")
			c.Println(a...)
			color.Output = output
			mutex.Unlock()
		default:
			mutex.Lock()
			c.Print(ProgramName() + ": ")
			c.Println(a...)
			mutex.Unlock()
		}
	}

	// printf - same as print, but takes a format specifier
	printf = func(c *color.Color, f string, a ...interface{}) {
		switch c {
		case themesDB[currThemeName].Debug:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Printf(f, a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Fatal:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <FATAL> ")
			c.Printf(f, a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Error:
			mutex.Lock()
			output := color.Output
			color.Output = os.Stderr
			c.Print(ProgramName() + ": <ERROR> ")
			c.Printf(f, a...)
			color.Output = output
			mutex.Unlock()
		default:
			mutex.Lock()
			c.Print(ProgramName() + ": ")
			c.Printf(f, a...)
			mutex.Unlock()
		}
	}
)

// SetTheme sets a color theme
func SetTheme(themeName string) error {
	if !IsValidTheme(themeName) {
		return iodine.New(fmt.Errorf("Unsupported theme name [%s]", themeName), nil)
	}

	mutex.Lock()

	// Just another additional precaution to completely disable color.
	// Color theme is also necessary, because it does other useful things like exit-on-fatal..
	switch currThemeName {
	case "nocolor":
		color.NoColor = true
	default:
		color.NoColor = false
	}

	currThemeName = themeName
	theme := themesDB[currThemeName]

	// Error prints a error message
	Error = func(a ...interface{}) { print(theme.Error, a...) }
	// Errorln prints a error message with a new line
	Errorln = func(a ...interface{}) { println(theme.Error, a...) }
	// Errorf prints a error message with formatting
	Errorf = func(f string, a ...interface{}) { printf(theme.Error, f, a...) }

	// Info prints a informational message
	Info = func(a ...interface{}) { print(theme.Info, a...) }
	// Infoln prints a informational message with a new line
	Infoln = func(a ...interface{}) { println(theme.Info, a...) }
	// Infof prints a informational message with formatting
	Infof = func(f string, a ...interface{}) { printf(theme.Info, f, a...) }

	// Debug prints a debug message
	Debug = func(a ...interface{}) {
		if !NoDebugPrint {
			print(theme.Debug, a...)
		}
	}
	// Debugln prints a debug message with a new line
	Debugln = func(a ...interface{}) {
		if !NoDebugPrint {
			println(theme.Debug, a...)
		}
	}
	// Debugf prints a debug message with formatting
	Debugf = func(f string, a ...interface{}) {
		if !NoDebugPrint {
			printf(theme.Debug, f, a...)
		}
	}

	Dir = theme.Dir.SprintfFunc()
	File = theme.File.SprintfFunc()
	Size = theme.Size.SprintfFunc()
	Time = theme.Time.SprintfFunc()
	Retry = theme.Retry.SprintfFunc()

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
