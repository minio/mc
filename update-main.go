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
	"time"

	"github.com/minio/mc/internal/github.com/minio/cli"
	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/pkg/console"
)

// Updates container to hold updates json
type Updates struct {
	BuildDate string
	Platforms map[string]string
}

const (
	mcUpdateURL = "https://dl.minio.io:9000/updates/updates.json"
)

// Help message.
var updateCmd = cli.Command{
	Name:   "update",
	Usage:  "Check for new software updates",
	Action: mainUpdate,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}}

EXAMPLES:
   1. Check for new updates
      $ mc update

`,
}

// mainUpdate -
func mainUpdate(ctx *cli.Context) {
	if ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}
	fatalIf(doUpdateCheck())
}

func doUpdateCheck() *probe.Error {
	clnt, err := url2Client(mcUpdateURL)
	if err != nil {
		return err.Trace(mcUpdateURL)
	}
	data, _, err := clnt.GetObject(0, 0)
	if err != nil {
		return err.Trace(mcUpdateURL)
	}
	current, _ := time.Parse(time.RFC3339Nano, Version)
	if current.IsZero() {
		console.Infoln(`Version is empty, must be a custom build cannot update. Please download releases from
https://dl.minio.io:9000 for continuous updates`)
		return nil
	}
	var updates Updates
	decoder := json.NewDecoder(data)
	if err := decoder.Decode(&updates); err != nil {
		return probe.NewError(err)
	}
	latest, _ := time.Parse(http.TimeFormat, updates.BuildDate)
	if latest.IsZero() {
		console.Infoln("No update available at this time")
		return nil
	}
	mcUpdateURLParse := clnt.URL()
	if latest.After(current) {
		var updateString string
		if runtime.GOOS == "windows" {
			updateString = "mc.exe cp " + mcUpdateURLParse.Scheme + "://" + mcUpdateURLParse.Host + string(mcUpdateURLParse.Separator) + updates.Platforms[runtime.GOOS] + " .\\mc.exe"
		} else {
			updateString = "mc cp " + mcUpdateURLParse.Scheme + "://" + mcUpdateURLParse.Host + string(mcUpdateURLParse.Separator) + updates.Platforms[runtime.GOOS] + " ./mc"
		}
		msg, err := printUpdateNotify(updateString, "new", "old")
		if err != nil {
			return probe.NewError(err)
		}
		console.Println(msg)
		return nil
	}
	console.Infoln("You are already running the most recent version of ‘mc’")
	return nil
}
