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
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
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
	Action: runUpdateCmd,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}}

EXAMPLES:
   1. Check for new updates
      $ mc update

`,
}

func doUpdateCheck(config *hostConfig) (string, error) {
	mcUpdateURLParse, err := client.Parse(mcUpdateURL)
	if err != nil {
		return "Unable to parse URL: " + mcUpdateURL, iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	clnt, err := getNewClient(mcUpdateURL, config)
	if err != nil {
		return "Unable to create client: " + mcUpdateURL, iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	data, _, err := clnt.GetObject(0, 0)
	if err != nil {
		return "Unable to read: " + mcUpdateURL, iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	current, _ := time.Parse(time.RFC3339Nano, Version)
	if current.IsZero() {
		message := `Version is empty, must be a custom build cannot update. Please download releases from
https://dl.minio.io:9000 for continuous updates`
		return message, nil
	}
	var updates Updates
	decoder := json.NewDecoder(data)
	err = decoder.Decode(&updates)
	if err != nil {
		return "Unable to parse update fields", iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	latest, _ := time.Parse(http.TimeFormat, updates.BuildDate)
	if latest.IsZero() {
		return "No update available at this time", nil
	}
	if latest.After(current) {
		updateString := "mc cp " + mcUpdateURLParse.Scheme + "://" + mcUpdateURLParse.Host + string(mcUpdateURLParse.Separator) + updates.Platforms[runtime.GOOS] + " ${HOME}/bin/mc"
		printUpdateNotify(updateString, "new", "old")
		return "", nil
	}
	return "You are already running the most recent version of ‘mc’", nil

}

// runUpdateCmd -
func runUpdateCmd(ctx *cli.Context) {
	if ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatals(ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errors.New("\"mc\" is not configured"), nil),
		})
	}
	hostConfig, err := getHostConfig(mcUpdateURL)
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unable to read configuration for host ‘%s’", mcUpdateURL),
			Error:   iodine.New(err, nil),
		})
	}
	msg, err := doUpdateCheck(hostConfig)
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: msg,
			Error:   iodine.New(err, nil),
		})
	}
	console.Infoln(msg)
}
