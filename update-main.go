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
	"errors"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// command specific flags.
var (
	updateFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help for update.",
		},
		cli.BoolFlag{
			Name:  "experimental, E",
			Usage: "Check experimental update.",
		},
	}
)

// Check for new software updates.
var updateCmd = cli.Command{
	Name:   "update",
	Usage:  "Check for a new software update.",
	Action: mainUpdate,
	Flags:  append(updateFlags, globalFlags...),
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Check for any new official release.
      $ mc {{.Name}}

   2. Check for any new experimental release.
      $ mc {{.Name}} --experimental
`,
}

// update URL endpoints.
const (
	mcUpdateStableURL       = "https://dl.minio.io/client/mc/release"
	mcUpdateExperimentalURL = "https://dl.minio.io/client/mc/experimental"
)

// updateMessage container to hold update messages.
type updateMessage struct {
	Status   string `json:"status"`
	Update   bool   `json:"update"`
	Download string `json:"downloadURL"`
	Version  string `json:"version"`
}

// String colorized update message.
func (u updateMessage) String() string {
	if !u.Update {
		return console.Colorize("Update", "You are already running the most recent version of ‘mc’.")
	}
	var msg string
	if runtime.GOOS == "windows" {
		msg = "Download " + u.Download
	} else {
		msg = "Download " + u.Download
	}
	msg, err := colorizeUpdateMessage(msg)
	fatalIf(err.Trace(msg), "Unable to colorize experimental update notification string ‘"+msg+"’.")
	return msg
}

// JSON jsonified update message.
func (u updateMessage) JSON() string {
	u.Status = "success"
	updateMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(updateMessageJSONBytes)
}

func parseReleaseData(data string) (time.Time, *probe.Error) {
	releaseStr := strings.Fields(data)
	if len(releaseStr) < 2 {
		return time.Time{}, probe.NewError(errors.New("Update data malformed"))
	}
	releaseDate := releaseStr[1]
	releaseDateSplits := strings.SplitN(releaseDate, ".", 3)
	if len(releaseDateSplits) < 3 {
		return time.Time{}, probe.NewError(errors.New("Update data malformed"))
	}
	if releaseDateSplits[0] != "mc" {
		return time.Time{}, probe.NewError(errors.New("Update data malformed, missing mc tag"))
	}
	// "OFFICIAL" tag is still kept for backward compatibility, we should remove this for the next release.
	if releaseDateSplits[1] != "RELEASE" && releaseDateSplits[1] != "OFFICIAL" {
		return time.Time{}, probe.NewError(errors.New("Update data malformed, missing RELEASE tag"))
	}
	dateSplits := strings.SplitN(releaseDateSplits[2], "T", 2)
	if len(dateSplits) < 2 {
		return time.Time{}, probe.NewError(errors.New("Update data malformed, not in modified RFC3359 form"))
	}
	dateSplits[1] = strings.Replace(dateSplits[1], "-", ":", -1)
	date := strings.Join(dateSplits, "T")

	parsedDate, e := time.Parse(time.RFC3339, date)
	if e != nil {
		return time.Time{}, probe.NewError(e)
	}
	return parsedDate, nil
}

// verify updates for releases.
func getReleaseUpdate(updateURL string) {
	// Construct a new update url.
	newUpdateURLPrefix := updateURL + "/" + runtime.GOOS + "-" + runtime.GOARCH
	newUpdateURL := newUpdateURLPrefix + "/mc.shasum"

	// Instantiate a new client with 3 sec timeout.
	client := &http.Client{
		Timeout: 3000 * time.Millisecond,
	}

	// Get the downloadURL.
	var downloadURL string
	switch runtime.GOOS {
	case "windows":
		// For windows and darwin.
		downloadURL = newUpdateURLPrefix + "/mc.exe"
	default:
		// For all other operating systems.
		downloadURL = newUpdateURLPrefix + "/mc"
	}

	data, e := client.Get(newUpdateURL)
	fatalIf(probe.NewError(e), "Unable to read from update URL ‘"+newUpdateURL+"’.")

	if strings.HasPrefix(mcVersion, "DEVELOPMENT.GOGET") {
		fatalIf(errDummy().Trace(newUpdateURL),
			"Update mechanism is not supported for ‘go get’ based binary builds.  Please download official releases from https://minio.io/#minio")
	}

	current, e := time.Parse(time.RFC3339, mcVersion)
	fatalIf(probe.NewError(e), "Unable to parse version string as time.")

	if current.IsZero() {
		fatalIf(errDummy().Trace(newUpdateURL),
			"Updates not supported for custom builds. Version field is empty. Please download official releases from https://minio.io/#minio")
	}

	body, e := ioutil.ReadAll(data.Body)
	fatalIf(probe.NewError(e), "Fetching updates failed. Please try again.")

	latest, err := parseReleaseData(string(body))
	fatalIf(err.Trace(newUpdateURL), "Please report this issue at https://github.com/minio/mc/issues.")

	if latest.IsZero() {
		fatalIf(errDummy().Trace(newUpdateURL),
			"Unable to validate any update available at this time. Please open an issue at https://github.com/minio/mc/issues")
	}

	updateMsg := updateMessage{
		Download: downloadURL,
		Version:  mcVersion,
	}
	if latest.After(current) {
		updateMsg.Update = true
	}
	printMsg(updateMsg)
}

// main entry point for update command.
func mainUpdate(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// Additional command speific theme customization.
	console.SetColor("Update", color.New(color.FgGreen, color.Bold))

	// Check for update.
	if ctx.Bool("experimental") {
		getReleaseUpdate(mcUpdateExperimentalURL)
	} else {
		getReleaseUpdate(mcUpdateStableURL)
	}
}
