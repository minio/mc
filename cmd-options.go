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
var commands = []cli.Command{
	lsCmd,     // list files and objects
	mbCmd,     // make a bucket
	catCmd,    // concantenate an object to standard output
	cpCmd,     // copy objects and files from multiple sources to single destination
	syncCmd,   // copy objects and files from single source to multiple destionations
	diffCmd,   // compare two objects
	accessCmd, // set permissions [public, private, readonly, authenticated] for buckets and folders.
	configCmd, // generate configuration "/home/harsha/.mc/config.json" file.
	updateCmd, // update Check for new software updates
	// Add your new commands starting from here
}

// Collection of mc flags currently supported
var (
	flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Supress chatty console output",
		},
		cli.BoolFlag{
			Name:  "alias",
			Usage: "Mimic operating system toolchain behavior wherever it makes sense",
		},
		cli.StringFlag{
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
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "Enable json formatted output",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable HTTP tracing",
		},
		// Add your new flags starting here
	}
)
