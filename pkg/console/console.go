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
	"github.com/minio/minio/pkg/iodine"
	"github.com/shiena/ansicolor"
)

// NoDebugPrint defines if the input should be printed in debug or not. By default it's set to true.
var NoDebugPrint = true

// NoJsonPrint defines if the input should be printed in json formatted or not. By default it's set to true.
var NoJSONPrint = true

// Message info string
type Message string

// ErrorMessage error message structure
type ErrorMessage struct {
	Message string `json:"Message"`
	Error   error  `json:"Reason"`
}

// Content content message structure
type Content struct {
	Filetype string `json:"ContentType"`
	Time     string `json:"LastModified"`
	Size     string `json:"Size"`
	Name     string `json:"Name"`
}

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

	// Print prints a message
	Print = themesDB[currThemeName].Info.Print

	// Fatal prints a error message and exits
	Fatal = func(msg ErrorMessage) {
		defer os.Exit(1)
		if msg.Error != nil {
			if NoJSONPrint {
				reason := "Reason: " + iodine.ToError(msg.Error).Error()
				message := msg.Message + ", " + reason
				print(themesDB[currThemeName].Error, message)
				if !NoDebugPrint {
					print(themesDB[currThemeName].Error, msg.Error)
				}
				return
			}
			errorMessageBytes, _ := json.Marshal(&msg)
			print(themesDB[currThemeName].JSON, string(errorMessageBytes))
		}
	}
	// Fatalln prints a error message with a new line and exits
	Fatalln = func(msg ErrorMessage) {
		defer os.Exit(1)
		if msg.Error != nil {
			if NoJSONPrint {
				reason := "Reason: " + iodine.ToError(msg.Error).Error()
				message := msg.Message + ", " + reason
				println(themesDB[currThemeName].Error, message)
				if !NoDebugPrint {
					println(themesDB[currThemeName].Error, msg.Error)
				}
				return
			}
			errorMessageBytes, _ := json.Marshal(&msg)
			println(themesDB[currThemeName].JSON, string(errorMessageBytes))
		}
	}

	// Error prints a error message
	Error = func(msg ErrorMessage) {
		if msg.Error != nil {
			if NoJSONPrint {
				reason := "Reason: " + iodine.ToError(msg.Error).Error()
				message := msg.Message + ", " + reason
				print(themesDB[currThemeName].Error, message)
				if !NoDebugPrint {
					print(themesDB[currThemeName].Error, msg.Error)
				}
				return
			}
			errorMessageBytes, _ := json.Marshal(&msg)
			print(themesDB[currThemeName].JSON, string(errorMessageBytes))
		}
	}
	// Errorln prints a error message with a new line
	Errorln = func(msg ErrorMessage) {
		if msg.Error != nil {
			if NoJSONPrint {
				reason := "Reason: " + iodine.ToError(msg.Error).Error()
				message := msg.Message + ", " + reason
				println(themesDB[currThemeName].Error, message)
				if !NoDebugPrint {
					println(themesDB[currThemeName].Error, msg.Error)
				}
				return
			}
			errorMessageBytes, _ := json.Marshal(&msg)
			println(themesDB[currThemeName].JSON, string(errorMessageBytes))
		}
	}

	// Info prints a informational message
	Info = func(msg Message) {
		if NoJSONPrint {
			print(themesDB[currThemeName].Info, msg)
			return
		}
		infoBytes, _ := json.Marshal(&msg)
		print(themesDB[currThemeName].JSON, string(infoBytes))
	}

	// Infoln prints a informational message with a new line
	Infoln = func(msg Message) {
		if NoJSONPrint {
			println(themesDB[currThemeName].Info, msg)
			return
		}
		infoBytes, _ := json.Marshal(&msg)
		println(themesDB[currThemeName].JSON, string(infoBytes))
	}

	// Debug prints a debug message without a new line
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
	// ContentInfo prints a structure Content
	ContentInfo = func(c Content) {
		if NoJSONPrint {
			print(themesDB[currThemeName].Time, c.Time)
			print(themesDB[currThemeName].Size, c.Size)
			switch c.Filetype {
			case "inode/directory":
				println(themesDB[currThemeName].Dir, c.Name)
			case "application/octet-stream":
				println(themesDB[currThemeName].File, c.Name)
			}
			return
		}
		contentBytes, _ := json.Marshal(&c)
		println(themesDB[currThemeName].JSON, string(contentBytes))
	}

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
		// special cases only for Content where it requires custom formatting
		case themesDB[currThemeName].Time:
			mutex.Lock()
			c.Print(fmt.Sprintf("[%s]", a...))
			mutex.Unlock()
		case themesDB[currThemeName].Size:
			mutex.Lock()
			c.Printf(fmt.Sprintf("%6s ", a...))
			mutex.Unlock()
		default:
			mutex.Lock()
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
		case themesDB[currThemeName].Dir:
			mutex.Lock()
			// ugly but its needed
			c.Printf("%s/\n", a...)
			mutex.Unlock()
		case themesDB[currThemeName].File:
			mutex.Lock()
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
