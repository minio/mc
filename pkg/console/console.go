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
	"sync"

	"path/filepath"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/minio/minio/pkg/probe"
	"github.com/shiena/ansicolor"
)

// NoDebugPrint defines if the input should be printed in debug or not. By default it's set to true.
var NoDebugPrint = true

// IsTesting this flag indicates if IsExited should be set or not, false by default
var IsTesting = false

// IsExited sets this boolean value if Fatal is called instead of os.Exit(1)
var IsExited = false

// Theme holds console color scheme
type Theme struct {
	Fatal     *color.Color
	Error     *color.Color
	Info      *color.Color
	Debug     *color.Color
	Size      *color.Color
	Time      *color.Color
	File      *color.Color
	Dir       *color.Color
	Command   *color.Color
	SessionID *color.Color
	JSON      *color.Color
	Bar       *color.Color
	PrintC    *color.Color
	Print     *color.Color
}

var (
	mutex = &sync.RWMutex{}

	stdoutColoredOutput = ansicolor.NewAnsiColorWriter(os.Stdout)
	stderrColoredOutput = ansicolor.NewAnsiColorWriter(os.Stderr)

	// themesDB contains supported list of Themes
	themesDB = map[string]Theme{
		"minimal": MiniTheme,
		"nocolor": NoColorTheme,
		"white":   WhiteTheme,
	}

	// currTheme is current theme
	currThemeName = func() string {
		theme := GetDefaultThemeName()
		// if not a TTY disable color
		if !isatty.IsTerminal(os.Stdout.Fd()) || !isatty.IsTerminal(os.Stderr.Fd()) {
			theme = "nocolor"
		}
		return theme
	}()

	// Bar print progress bar
	Bar = func(data ...interface{}) {
		print(themesDB[currThemeName].Bar, data...)
	}

	// Print prints a message
	Print = func(data ...interface{}) {
		print(themesDB[currThemeName].Print, data...)
		return
	}

	// PrintC prints a message with color
	PrintC = func(data ...interface{}) {
		print(themesDB[currThemeName].PrintC, data...)
		return
	}

	// Printf prints a formatted message
	Printf = func(f string, data ...interface{}) {
		printf(themesDB[currThemeName].Print, f, data...)
		return
	}

	// Println prints a message with a newline
	Println = func(data ...interface{}) {
		println(themesDB[currThemeName].Print, data...)
	}

	// Fatal print a error message and exit
	Fatal = func(data ...interface{}) {
		print(themesDB[currThemeName].Fatal, data...)
		if !IsTesting {
			os.Exit(1)
		}
		defer func() {
			IsExited = true
		}()
		return
	}

	// Fatalf print a error message with a format specified and exit
	Fatalf = func(f string, data ...interface{}) {
		printf(themesDB[currThemeName].Fatal, f, data...)
		if !IsTesting {
			os.Exit(1)
		}
		defer func() {
			IsExited = true
		}()
		return
	}

	// Fatalln print a error message with a new line and exit
	Fatalln = func(data ...interface{}) {
		println(themesDB[currThemeName].Fatal, data...)
		if !IsTesting {
			os.Exit(1)
		}
		defer func() {
			IsExited = true
		}()
		return
	}

	// Error prints a error message
	Error = func(data ...interface{}) {
		if IsTesting {
			defer func() {
				IsExited = true
			}()
		}
		print(themesDB[currThemeName].Error, data...)
		return
	}

	// Errorf print a error message with a format specified
	Errorf = func(f string, data ...interface{}) {
		if IsTesting {
			defer func() {
				IsExited = true
			}()
		}
		printf(themesDB[currThemeName].Error, f, data...)
		return
	}

	// Errorln prints a error message with a new line
	Errorln = func(data ...interface{}) {
		if IsTesting {
			defer func() {
				IsExited = true
			}()
		}
		println(themesDB[currThemeName].Error, data...)
		return
	}

	// Info prints a informational message
	Info = func(data ...interface{}) {
		print(themesDB[currThemeName].Info, data...)
		return
	}

	// Infof prints a informational message in custom format
	Infof = func(f string, data ...interface{}) {
		printf(themesDB[currThemeName].Info, f, data...)
		return
	}

	// Infoln prints a informational message with a new line
	Infoln = func(data ...interface{}) {
		println(themesDB[currThemeName].Info, data...)
		return
	}

	// Debug prints a debug message without a new line
	// Debug prints a debug message
	Debug = func(data ...interface{}) {
		if !NoDebugPrint {
			print(themesDB[currThemeName].Debug, data...)
		}
	}

	// Debugf prints a debug message with a new line
	Debugf = func(f string, data ...interface{}) {
		if !NoDebugPrint {
			printf(themesDB[currThemeName].Debug, f, data...)
		}
	}

	// Debugln prints a debug message with a new line
	Debugln = func(data ...interface{}) {
		if !NoDebugPrint {
			println(themesDB[currThemeName].Debug, data...)
		}
	}

	// Time helper to print Time theme
	Time = themesDB[currThemeName].Time.SprintfFunc()
	// Size helper to print Size theme
	Size = themesDB[currThemeName].Size.SprintfFunc()
	// File helper to print File theme
	File = themesDB[currThemeName].File.SprintfFunc()
	// Dir helper to print Dir theme
	Dir = themesDB[currThemeName].Dir.SprintfFunc()
	// Command helper to print command theme
	Command = themesDB[currThemeName].Command.SprintfFunc()
	// SessionID helper to print sessionid theme
	SessionID = themesDB[currThemeName].SessionID.SprintfFunc()
	// JSON helper to print json strings
	JSON = themesDB[currThemeName].JSON.SprintfFunc()
)

var (
	// wrap around standard fmt functions
	// print prints a message prefixed with message type and program name
	print = func(c *color.Color, a ...interface{}) {
		switch c {
		case themesDB[currThemeName].Debug:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Print(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Fatal:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <ERROR> ")
			c.Print(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Error:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <ERROR> ")
			c.Print(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Info:
			mutex.Lock()
			c.Print(ProgramName() + ": ")
			c.Print(a...)
			mutex.Unlock()
		default:
			mutex.Lock()
			c.Print(a...)
			mutex.Unlock()
		}
	}

	// printf - same as print with a new line
	printf = func(c *color.Color, f string, a ...interface{}) {
		switch c {
		case themesDB[currThemeName].Debug:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Printf(f, a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Fatal:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <ERROR> ")
			c.Printf(f, a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Error:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <ERROR> ")
			c.Printf(f, a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Info:
			mutex.Lock()
			c.Print(ProgramName() + ": ")
			c.Printf(f, a...)
			mutex.Unlock()
		default:
			mutex.Lock()
			c.Printf(f, a...)
			mutex.Unlock()
		}
	}

	// println - same as print with a new line
	println = func(c *color.Color, a ...interface{}) {
		switch c {
		case themesDB[currThemeName].Debug:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <DEBUG> ")
			c.Println(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Fatal:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <ERROR> ")
			c.Println(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Error:
			mutex.Lock()
			output := color.Output
			color.Output = stderrColoredOutput
			c.Print(ProgramName() + ": <ERROR> ")
			c.Println(a...)
			color.Output = output
			mutex.Unlock()
		case themesDB[currThemeName].Info:
			mutex.Lock()
			c.Print(ProgramName() + ": ")
			c.Println(a...)
			mutex.Unlock()
		default:
			mutex.Lock()
			c.Println(a...)
			mutex.Unlock()
		}
	}
)

// Lock console
func Lock() {
	mutex.Lock()
}

// Unlock locked console
func Unlock() {
	mutex.Unlock()
}

// SetTheme sets a color theme
func SetTheme(themeName string) error {
	if !IsValidTheme(themeName) {
		return probe.New(fmt.Errorf("Unsupported theme name [%s]", themeName))
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

// ProgramName - return the name of the executable program
func ProgramName() string {
	_, progName := filepath.Split(os.Args[0])
	return progName
}
