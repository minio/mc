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

package main

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Check for new software updates.
var updateCmd = cli.Command{
	Name:   "update",
	Usage:  "Check for new software updates.",
	Action: mainUpdate,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}}

EXAMPLES:
   1. Check for new official releases
      $ mc {{.Name}} release

   2. Check for new experimental releases
      $ mc {{.Name}} experimental
`,
}

// Updates container to hold updates json
type Updates struct {
	BuildDate string
	Platforms map[string]string
}

// UpdateMessage container to hold update messages
type UpdateMessage struct {
	Update   bool      `json:"update"`
	Download string    `json:"download-url"`
	Version  time.Time `json:"version"`
}

func (u UpdateMessage) String() string {
	if !globalJSONFlag {
		if u.Update {
			var msg string
			if runtime.GOOS == "windows" {
				msg = "mc.exe cp " + u.Download + " .\\mc.exe"
			} else {
				msg = "mc cp " + u.Download + " ./mc.new; chmod 755 ./mc.new"
			}
			msg, err := colorizeUpdateMessage(msg)
			fatalIf(err.Trace(msg), "Unable to colorize experimental update notification string ‘"+msg+"’.")
			return msg
		}
		return console.Colorize("UpdateMessage", "You are already running the most recent version of ‘mc’.")
	}
	updateMessageJSONBytes, err := json.Marshal(u)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(updateMessageJSONBytes)
}

const (
	mcUpdateURL       = "https://dl.minio.io:9000/updates/updates.json"
	mcExperimentalURL = "https://dl.minio.io:9000/updates/experimental.json"
)

func checkUpdateSyntax(ctx *cli.Context) {
	if ctx.Args().First() == "help" || !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}
	arg := strings.TrimSpace(ctx.Args().First())
	if arg != "release" && arg != "experimental" {
		fatalIf(errInvalidArgument().Trace(arg), "Unrecognized argument provided.")
	}
}

// mainUpdate -
func mainUpdate(ctx *cli.Context) {
	checkUpdateSyntax(ctx)

	console.SetCustomTheme(map[string]*color.Color{
		"UpdateMessage": color.New(color.FgGreen, color.Bold),
	})

	arg := strings.TrimSpace(ctx.Args().First())
	switch arg {
	case "release":
		getReleaseUpdate()
	case "experimental":
		getExperimentalUpdate()
	}
}

func getExperimentalUpdate() {
	clnt, err := url2Client(mcExperimentalURL)
	fatalIf(err.Trace(mcExperimentalURL), "Unable to initalize experimental URL.")

	data, _, err := clnt.GetObject(0, 0)
	fatalIf(err.Trace(mcExperimentalURL), "Unable to read from experimental URL ‘"+mcExperimentalURL+"’.")

	current, e := time.Parse(time.RFC3339Nano, Version)
	fatalIf(probe.NewError(e), "Unable to parse Version string as time.")

	if current.IsZero() {
		fatalIf(errDummy().Trace(), "Experimental updates not supported for custom build. Version field is empty. Please download official releases from https://dl.minio.io:9000")
	}

	var experimentals Updates
	decoder := json.NewDecoder(data)
	e = decoder.Decode(&experimentals)
	fatalIf(probe.NewError(e), "Unable to decode experimental update notification.")

	latest, e := time.Parse(http.TimeFormat, experimentals.BuildDate)
	fatalIf(probe.NewError(e), "Unable to parse BuildDate.")

	if latest.IsZero() {
		fatalIf(errDummy().Trace(), "Unable to validate any experimental update available at this time. Please open an issue at https://github.com/minio/mc/issues")
	}

	mcExperimentalURLParse := clnt.URL()
	if latest.After(current) {
		console.Println(UpdateMessage{
			Update:   true,
			Download: mcExperimentalURLParse.Scheme + "://" + mcExperimentalURLParse.Host + string(mcExperimentalURLParse.Separator) + experimentals.Platforms[runtime.GOOS],
			Version:  latest,
		})
		return
	}
	console.Println(UpdateMessage{
		Update:   false,
		Download: mcExperimentalURLParse.Scheme + "://" + mcExperimentalURLParse.Host + string(mcExperimentalURLParse.Separator) + experimentals.Platforms[runtime.GOOS],
		Version:  latest,
	})
}

func getReleaseUpdate() {
	clnt, err := url2Client(mcUpdateURL)
	fatalIf(err.Trace(mcUpdateURL), "Unable to initalize update URL.")

	data, _, err := clnt.GetObject(0, 0)
	fatalIf(err.Trace(mcUpdateURL), "Unable to read from update URL ‘"+mcUpdateURL+"’.")

	current, e := time.Parse(time.RFC3339Nano, Version)
	fatalIf(probe.NewError(e), "Unable to parse Version string as time.")

	if current.IsZero() {
		fatalIf(errDummy().Trace(), "Updates not supported for custom build. Version field is empty. Please download official releases from https://dl.minio.io:9000")
	}

	var updates Updates
	decoder := json.NewDecoder(data)
	e = decoder.Decode(&updates)
	fatalIf(probe.NewError(e), "Unable to decode update notification.")

	latest, e := time.Parse(http.TimeFormat, updates.BuildDate)
	fatalIf(probe.NewError(e), "Unable to parse BuildDate.")

	if latest.IsZero() {
		fatalIf(errDummy().Trace(), "Unable to validate any update available at this time. Please open an issue at https://github.com/minio/mc/issues")
	}

	mcUpdateURLParse := clnt.URL()
	if latest.After(current) {
		console.Println(UpdateMessage{
			Update:   true,
			Download: mcUpdateURLParse.Scheme + "://" + mcUpdateURLParse.Host + string(mcUpdateURLParse.Separator) + updates.Platforms[runtime.GOOS],
			Version:  latest,
		})
		return
	}
	console.Println(UpdateMessage{
		Update:   false,
		Download: mcUpdateURLParse.Scheme + "://" + mcUpdateURLParse.Host + string(mcUpdateURLParse.Separator) + updates.Platforms[runtime.GOOS],
		Version:  latest,
	})
	return
}
