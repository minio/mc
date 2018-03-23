/*
 * Minio Client (C) 2015, 2016, 2017 Minio, Inc.
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

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// Check for new software updates.
var updateCmd = cli.Command{
	Name:   "update",
	Usage:  "Check for a new software update.",
	Action: mainUpdate,
	Before: setGlobalsFromContext,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Suppress chatty console output.",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "Enable JSON formatted output.",
		},
	},
	CustomHelpTemplate: `Name:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}}{{if .VisibleFlags}} [FLAGS]{{end}}
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
   1. Check if there is a new update available:
       $ {{.HelpName}}
`,
}

// updateMessage container to hold update messages.
type updateMessage struct {
	Status    string `json:"status"`
	Update    bool   `json:"update"`
	Download  string `json:"downloadURL,omitempty"`
	Version   string `json:"version"`
	olderThan time.Duration
}

// String colorized update message.
func (u updateMessage) String() string {
	if u.olderThan == time.Duration(0) {
		return console.Colorize("Update", "You are already running the most recent version of `mc`.")
	}
	return colorizeUpdateMessage(u.Download, u.olderThan)
}

// JSON jsonified update message.
func (u updateMessage) JSON() string {
	u.Status = "success"
	u.Update = u.olderThan != time.Duration(0)
	updateMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(updateMessageJSONBytes)
}

const releaseTagTimeLayout = "2006-01-02T15-04-05Z"
const mcReleaseURL = "https://dl.minio.io/client/mc/release/" + runtime.GOOS + "-" + runtime.GOARCH + "/"

func getCurrentReleaseTime(mcVersion, mcBinaryPath string) (releaseTime time.Time, err *probe.Error) {
	var e error
	if releaseTime, e = time.Parse(time.RFC3339, mcVersion); e == nil {
		return releaseTime, probe.NewError(e)
	}

	mcBinaryPath, e = exec.LookPath(mcBinaryPath)
	if e != nil {
		return releaseTime, probe.NewError(e)
	}

	fi, e := os.Stat(mcBinaryPath)
	if e != nil {
		e = fmt.Errorf("Unable to get ModTime of %s. %s", mcBinaryPath, e)
	} else {
		releaseTime = fi.ModTime().UTC()
	}

	return releaseTime, probe.NewError(e)
}

// GetCurrentReleaseTime - returns this process's release time.  If it is official minio version,
// parsed version is returned else minio binary's mod time is returned.
func GetCurrentReleaseTime() (releaseTime time.Time, err *probe.Error) {
	return getCurrentReleaseTime(Version, os.Args[0])
}

func isDocker(cgroupFile string) (bool, *probe.Error) {
	cgroup, e := ioutil.ReadFile(cgroupFile)
	if os.IsNotExist(e) {
		e = nil
	}

	return bytes.Contains(cgroup, []byte("docker")), probe.NewError(e)
}

// IsDocker - returns if the environment is docker or not.
func IsDocker() bool {
	found, err := isDocker("/proc/self/cgroup")
	fatalIf(err.Trace("/proc/self/cgroup"), "Unable to validate if this is a docker container.")
	return found
}

func isSourceBuild(mcVersion string) bool {
	_, e := time.Parse(time.RFC3339, mcVersion)
	return e != nil
}

// IsSourceBuild - returns if this binary is made from source or not.
func IsSourceBuild() bool {
	return isSourceBuild(Version)
}

// DO NOT CHANGE USER AGENT STYLE.
// The style should be
//   Minio (<OS>; <ARCH>[; docker][; source])  mc/<VERSION> mc/<RELEASE-TAG> mc/<COMMIT-ID>
//
// For any change here should be discussed by openning an issue at https://github.com/minio/mc/issues.
func getUserAgent() string {
	userAgent := "Minio (" + runtime.GOOS + "; " + runtime.GOARCH
	if IsDocker() {
		userAgent += "; docker"
	}
	if IsSourceBuild() {
		userAgent += "; source"
	}
	userAgent += ") " + " mc/" + Version + " mc/" + ReleaseTag + " mc/" + CommitID

	return userAgent
}

func downloadReleaseData(releaseChecksumURL string, timeout time.Duration) (data string, err *probe.Error) {
	req, e := http.NewRequest("GET", releaseChecksumURL, nil)
	if e != nil {
		return data, probe.NewError(e)
	}
	req.Header.Set("User-Agent", getUserAgent())

	client := &http.Client{
		Timeout: timeout,
	}

	resp, e := client.Do(req)
	if err != nil {
		return data, probe.NewError(e)
	}
	if resp == nil {
		return data, probe.NewError(fmt.Errorf("No response from server to download URL %s", releaseChecksumURL))
	}

	if resp.StatusCode != http.StatusOK {
		return data, probe.NewError(fmt.Errorf("Error downloading URL %s. Response: %v", releaseChecksumURL, resp.Status))
	}

	dataBytes, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		return data, probe.NewError(fmt.Errorf("Error reading response. %s", e))
	}

	data = string(dataBytes)
	return data, nil
}

// DownloadReleaseData - downloads release data from minio official server.
func DownloadReleaseData(timeout time.Duration) (data string, err *probe.Error) {
	return downloadReleaseData(mcReleaseURL+"mc.sha256sum", timeout)
}

func parseReleaseData(data string) (releaseTime time.Time, err *probe.Error) {
	fields := strings.Fields(data)
	if len(fields) != 2 {
		e := fmt.Errorf("Unknown release data `%s`", data)
		return releaseTime, probe.NewError(e)
	}

	releaseInfo := fields[1]
	if fields = strings.Split(releaseInfo, "."); len(fields) != 3 {
		e := fmt.Errorf("Unknown release information `%s`", releaseInfo)
		return releaseTime, probe.NewError(e)
	}

	if !(fields[0] == "mc" && fields[1] == "RELEASE") {
		e := fmt.Errorf("Unknown release '%s'", releaseInfo)
		return releaseTime, probe.NewError(e)
	}

	var e error
	releaseTime, e = time.Parse(releaseTagTimeLayout, fields[2])
	if e != nil {
		e = fmt.Errorf("Unknown release time format. %s", err)
	}

	return releaseTime, probe.NewError(e)
}

func getLatestReleaseTime(timeout time.Duration) (releaseTime time.Time, err *probe.Error) {
	data, err := DownloadReleaseData(timeout)
	if err != nil {
		return releaseTime, err.Trace()
	}
	return parseReleaseData(data)
}

func getDownloadURL() (downloadURL string) {
	if IsDocker() {
		return "docker pull minio/mc"
	}

	if runtime.GOOS == "windows" {
		return mcReleaseURL + "mc.exe"
	}

	return mcReleaseURL + "mc"
}

func getUpdateInfo(timeout time.Duration) (older time.Duration, downloadURL string, err *probe.Error) {
	currentReleaseTime, err := GetCurrentReleaseTime()
	if err != nil {
		return older, downloadURL, err.Trace()
	}

	latestReleaseTime, err := getLatestReleaseTime(timeout)
	if err != nil {
		return older, downloadURL, err.Trace()
	}

	if latestReleaseTime.After(currentReleaseTime) {
		older = latestReleaseTime.Sub(currentReleaseTime)
		downloadURL = getDownloadURL()
	}

	return older, downloadURL, nil
}

func mainUpdate(ctx *cli.Context) {
	if len(ctx.Args()) != 0 {
		cli.ShowCommandHelpAndExit(ctx, "update", -1)
	}

	// Additional command speific theme customization.
	console.SetColor("Update", color.New(color.FgGreen, color.Bold))

	older, downloadURL, err := getUpdateInfo(10 * time.Second)
	fatalIf(err.Trace(downloadURL), "Unable to fetch update info for mc.")

	if !globalQuiet {
		printMsg(updateMessage{
			olderThan: older,
			Download:  downloadURL,
			Version:   Version,
		})
	}
}
