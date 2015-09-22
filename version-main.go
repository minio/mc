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
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Print version.
var versionCmd = cli.Command{
	Name:   "version",
	Usage:  "Print version.",
	Action: mainVersion,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}

`,
}

func mainVersion(ctx *cli.Context) {
	if ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "version", 1) // last argument is exit code
	}
	console.SetCustomTheme(map[string]*color.Color{
		"Version": color.New(color.FgGreen, color.Bold),
	})
	if globalJSONFlag {
		tB, e := json.Marshal(
			struct {
				Version struct {
					Value  string `json:"value"`
					Format string `json:"format"`
				} `json:"version"`
			}{
				Version: struct {
					Value  string `json:"value"`
					Format string `json:"format"`
				}{
					Value:  mcVersion,
					Format: "RFC2616",
				},
			},
		)
		fatalIf(probe.NewError(e), "Unable to construct version string.")
		console.Println(string(tB))
		return
	}
	msg := console.Colorize("Version", fmt.Sprintf("Version: %s\n", mcVersion))
	msg += console.Colorize("Version", fmt.Sprintf("Release-Tag: %s", mcReleaseTag))
	console.Println(msg)
}
