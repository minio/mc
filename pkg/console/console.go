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
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"path/filepath"

	"github.com/fatih/color"
	"github.com/fatih/structs"
	"github.com/minio/minio/pkg/iodine"
	"github.com/shiena/ansicolor"
)

// NoDebugPrint defines if the input should be printed in debug or not. By default it's set to true.
var NoDebugPrint = true

// NoJsonPrint defines if the input should be printed in json formatted or not. By default it's set to true.
var NoJSONPrint = true

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
	JSON  *color.Color
	Bar   *color.Color
	Print *color.Color
}

func readErrorFromdata(data interface{}) error {
	st := structs.New(data)
	if st.IsZero() {
		return nil
	}
	msgErr := st.Field("Error")
	return msgErr.Value().(error)
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
		if !isatty(os.Stdout.Fd()) {
			return "nocolor"
		}
		return theme
	}()

	// Bar print progress bar
	Bar = themesDB[currThemeName].Info.Print

	// Print prints a message
	Print = func(data ...interface{}) {
		if NoJSONPrint {
			print(themesDB[currThemeName].Print, data...)
			return
		}
		for i := 0; i < len(data); i++ {
			printBytes, _ := json.Marshal(&data[i])
			print(themesDB[currThemeName].JSON, string(printBytes))
		}
	}

	// Printf prints a formatted message
	Printf = func(f string, data ...interface{}) {
		if NoJSONPrint {
			printf(themesDB[currThemeName].Print, f, data...)
			return
		}
		for i := 0; i < len(data); i++ {
			printBytes, _ := json.Marshal(&data[i])
			printf(themesDB[currThemeName].JSON, "", string(printBytes))
		}
	}

	// Println prints a message with a newline
	Println = func(data ...interface{}) {
		if NoJSONPrint {
			println(themesDB[currThemeName].Print, data...)
			return
		}
		for i := 0; i < len(data); i++ {
			printBytes, _ := json.Marshal(&data[i])
			println(themesDB[currThemeName].JSON, string(printBytes))
		}
	}

	// Fatal prints a error message and exits
	Fatal = func(data ...interface{}) {
		defer os.Exit(1)
		for i := 0; i < len(data); i++ {
			err := readErrorFromdata(data[i])
			if err != nil {
				if NoJSONPrint {
					print(themesDB[currThemeName].Error, data[i])
					if !NoDebugPrint {
						print(themesDB[currThemeName].Error, err)
					}
					return
				}
				errorMessageBytes, _ := json.Marshal(&data[i])
				print(themesDB[currThemeName].JSON, string(errorMessageBytes))
			}
		}
	}

	// Fatalf is undefined since under the context of typed messages formatting
	// is already done before caller invokes console.Fatal*

	// Fatalln prints a error message with a new line and exits
	Fatalln = func(data ...interface{}) {
		defer os.Exit(1)
		for i := 0; i < len(data); i++ {
			err := readErrorFromdata(data[i])
			if err != nil {
				if NoJSONPrint {
					println(themesDB[currThemeName].Error, data[i])
					if !NoDebugPrint {
						println(themesDB[currThemeName].Error, err)
					}
					return
				}
				errorMessageBytes, _ := json.Marshal(&data[i])
				println(themesDB[currThemeName].JSON, string(errorMessageBytes))
			}
		}
	}

	// Error prints a error message
	Error = func(data ...interface{}) {
		for i := 0; i < len(data); i++ {
			err := readErrorFromdata(data[i])
			if err != nil {
				if NoJSONPrint {
					print(themesDB[currThemeName].Error, data[i])
					if !NoDebugPrint {
						print(themesDB[currThemeName].Error, err)
					}
					return
				}
				errorMessageBytes, _ := json.Marshal(&data[i])
				print(themesDB[currThemeName].JSON, string(errorMessageBytes))
			}
		}
	}

	// Errorf is undefined since under the context of typed messages formatting
	// is already done before caller invokes console.Error*

	// Errorln prints a error message with a new line
	Errorln = func(data ...interface{}) {
		for i := 0; i < len(data); i++ {
			err := readErrorFromdata(data[i])
			if err != nil {
				if NoJSONPrint {
					println(themesDB[currThemeName].Error, data[i])
					if !NoDebugPrint {
						println(themesDB[currThemeName].Error, err)
					}
					return
				}
				errorMessageBytes, _ := json.Marshal(&data[i])
				println(themesDB[currThemeName].JSON, string(errorMessageBytes))
			}
		}
	}

	// Info prints a informational message
	Info = func(data ...interface{}) {
		if NoJSONPrint {
			print(themesDB[currThemeName].Info, data...)
			return
		}
		for i := 0; i < len(data); i++ {
			infoBytes, _ := json.Marshal(&data[i])
			print(themesDB[currThemeName].JSON, string(infoBytes))
		}
	}

	// Infof prints a informational message in custom format
	Infof = func(f string, data ...interface{}) {
		if NoJSONPrint {
			printf(themesDB[currThemeName].Info, f, data...)
			return
		}
		for i := 0; i < len(data); i++ {
			infoBytes, _ := json.Marshal(&data[i])
			printf(themesDB[currThemeName].JSON, "", string(infoBytes))
		}
	}

	// Infoln prints a informational message with a new line
	Infoln = func(data ...interface{}) {
		if NoJSONPrint {
			println(themesDB[currThemeName].Info, data...)
			return
		}
		for i := 0; i < len(data); i++ {
			infoBytes, _ := json.Marshal(&data[i])
			println(themesDB[currThemeName].JSON, string(infoBytes))
		}
	}

	// Debug prints a debug message without a new line
	// Debug prints a debug message
	Debug = func(a ...interface{}) {
		if !NoDebugPrint {
			print(themesDB[currThemeName].Debug, a...)
		}
	}

	// Debugf prints a debug message with a new line
	Debugf = func(f string, a ...interface{}) {
		if !NoDebugPrint {
			printf(themesDB[currThemeName].Debug, f, a...)
		}
	}

	// Debugln prints a debug message with a new line
	Debugln = func(a ...interface{}) {
		if !NoDebugPrint {
			println(themesDB[currThemeName].Debug, a...)
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

	// Retry prints a retry message
	Retry = func(a ...interface{}) {
		println(themesDB[currThemeName].Retry, a...)
	}
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
			c.Print(ProgramName() + ": <FATAL> ")
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
			c.Print(ProgramName() + ": <FATAL> ")
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
