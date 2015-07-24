/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"fmt"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

// Collection of mc commands currently supported are
var commands = []cli.Command{}

// Collection of mc flags currently supported
var flags = []cli.Flag{}

var (
	configFlag = cli.StringFlag{
		Name:  "config, C",
		Value: mustGetMcConfigDir(),
		Usage: "Path to configuration folder",
	}

	quietFlag = cli.BoolFlag{
		Name:  "quiet, q",
		Usage: "Suppress chatty console output",
	}

	forceFlag = cli.BoolFlag{
		Name:  "force",
		Usage: "Force copying when destination exists",
	}

	aliasFlag = cli.BoolFlag{
		Name:  "alias",
		Usage: "Mimic operating system toolchain behavior wherever it makes sense",
	}

	themeFlag = cli.StringFlag{
		Name:  "theme",
		Value: console.GetDefaultThemeName(),
		Usage: fmt.Sprintf("Choose a console theme from this list [%s]", func() string {
			keys := []string{}
			for _, themeName := range console.GetThemeNames() {
				if console.GetThemeName() == themeName {
					themeName = "*" + themeName + "*"
				}
				keys = append(keys, themeName)
			}
			return strings.Join(keys, ", ")
		}()),
	}

	jsonFlag = cli.BoolFlag{
		Name:  "json",
		Usage: "Enable json formatted output",
	}

	debugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "Enable debugging output",
	}

	// Add your new flags starting here
)

// registerCmd registers a cli command
func registerCmd(cmd cli.Command) {
	commands = append(commands, cmd)
}

// registerFlag registers a cli flag
func registerFlag(flag cli.Flag) {
	flags = append(flags, flag)
}
