// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	isatty "github.com/mattn/go-isatty"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	_ "github.com/minio/sha256-simd" // Needed for sha256 hash verifier.
)

// Check for new software updates.
var updateCmd = cli.Command{
	Name:         "update",
	Usage:        "update mc to latest release",
	Action:       mainUpdate,
	OnUsageError: onUsageError,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "json",
			Usage: "enable JSON lines formatted output",
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
EXIT STATUS:
  0 - you are already running the most recent version
  1 - new update was applied successfully
 -1 - error in getting update information

EXAMPLES:
  1. Check and update mc:
     {{.Prompt}} {{.HelpName}}
`,
}

const (
	mcReleaseTagTimeLayout = "2006-01-02T15-04-05Z"
	mcOSARCH               = runtime.GOOS + "-" + runtime.GOARCH
	mcReleaseURL           = "https://dl.min.io/client/mc/release/" + mcOSARCH + "/"
)

var (
	// For windows our files have .exe additionally.
	mcReleaseWindowsInfoURL = mcReleaseURL + "mc.exe.sha256sum"
)

// mcVersionToReleaseTime - parses a standard official release
// mc --version string.
//
// An official binary's version string is the release time formatted
// with RFC3339 (in UTC) - e.g. `2017-09-29T19:16:56Z`
func mcVersionToReleaseTime(version string) (releaseTime time.Time, err *probe.Error) {
	var e error
	releaseTime, e = time.Parse(time.RFC3339, version)
	return releaseTime, probe.NewError(e)
}

// releaseTimeToReleaseTag - converts a time to a string formatted as
// an official mc release tag.
//
// An official mc release tag looks like:
// `RELEASE.2017-09-29T19-16-56Z`
func releaseTimeToReleaseTag(releaseTime time.Time) string {
	return "RELEASE." + releaseTime.Format(mcReleaseTagTimeLayout)
}

// releaseTagToReleaseTime - reverse of `releaseTimeToReleaseTag()`
func releaseTagToReleaseTime(releaseTag string) (releaseTime time.Time, err *probe.Error) {
	fields := strings.Split(releaseTag, ".")
	if len(fields) < 2 || len(fields) > 3 {
		return releaseTime, probe.NewError(fmt.Errorf("%s is not a valid release tag", releaseTag))
	}
	if fields[0] != "RELEASE" {
		return releaseTime, probe.NewError(fmt.Errorf("%s is not a valid release tag", releaseTag))
	}
	var e error
	releaseTime, e = time.Parse(mcReleaseTagTimeLayout, fields[1])
	return releaseTime, probe.NewError(e)
}

// getModTime - get the file modification time of `path`
func getModTime(path string) (t time.Time, err *probe.Error) {
	var e error
	path, e = filepath.EvalSymlinks(path)
	if e != nil {
		return t, probe.NewError(fmt.Errorf("Unable to get absolute path of %s. %w", path, e))
	}

	// Version is mc non-standard, we will use mc binary's
	// ModTime as release time.
	var fi os.FileInfo
	fi, e = os.Stat(path)
	if e != nil {
		return t, probe.NewError(fmt.Errorf("Unable to get ModTime of %s. %w", path, e))
	}

	// Return the ModTime
	return fi.ModTime().UTC(), nil
}

// GetCurrentReleaseTime - returns this process's release time.  If it
// is official mc --version, parsed version is returned else mc
// binary's mod time is returned.
func GetCurrentReleaseTime() (releaseTime time.Time, err *probe.Error) {
	if releaseTime, err = mcVersionToReleaseTime(Version); err == nil {
		return releaseTime, nil
	}

	// Looks like version is mc non-standard, we use mc
	// binary's ModTime as release time:
	path, e := os.Executable()
	if e != nil {
		return releaseTime, probe.NewError(e)
	}
	return getModTime(path)
}

// IsDocker - returns if the environment mc is running in docker or
// not. The check is a simple file existence check.
//
// https://github.com/moby/moby/blob/master/daemon/initlayer/setup_unix.go#L25
//
//     "/.dockerenv":      "file",
//
func IsDocker() bool {
	_, e := os.Stat("/.dockerenv")
	if os.IsNotExist(e) {
		return false
	}

	return e == nil
}

// IsDCOS returns true if mc is running in DCOS.
func IsDCOS() bool {
	// http://mesos.apache.org/documentation/latest/docker-containerizer/
	// Mesos docker containerizer sets this value
	return os.Getenv("MESOS_CONTAINER_NAME") != ""
}

// IsKubernetes returns true if MinIO is running in kubernetes.
func IsKubernetes() bool {
	// Kubernetes env used to validate if we are
	// indeed running inside a kubernetes pod
	// is KUBERNETES_SERVICE_HOST but in future
	// we might need to enhance this.
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// IsSourceBuild - returns if this binary is a non-official build from
// source code.
func IsSourceBuild() bool {
	_, err := mcVersionToReleaseTime(Version)
	return err != nil
}

// DO NOT CHANGE USER AGENT STYLE.
// The style should be
//
//   mc (<OS>; <ARCH>[; dcos][; kubernetes][; docker][; source]) mc/<VERSION> mc/<RELEASE-TAG> mc/<COMMIT-ID>
//
// Any change here should be discussed by opening an issue at
// https://github.com/minio/mc/issues.
func getUserAgent() string {

	userAgentParts := []string{}
	// Helper function to concisely append a pair of strings to a
	// the user-agent slice.
	uaAppend := func(p, q string) {
		userAgentParts = append(userAgentParts, p, q)
	}

	uaAppend("mc (", runtime.GOOS)
	uaAppend("; ", runtime.GOARCH)
	if IsDCOS() {
		uaAppend("; ", "dcos")
	}
	if IsKubernetes() {
		uaAppend("; ", "kubernetes")
	}
	if IsDocker() {
		uaAppend("; ", "docker")
	}
	if IsSourceBuild() {
		uaAppend("; ", "source")
	}

	uaAppend(") mc/", Version)
	uaAppend(" mc/", ReleaseTag)
	uaAppend(" mc/", CommitID)

	return strings.Join(userAgentParts, "")
}

func downloadReleaseURL(releaseChecksumURL string, timeout time.Duration) (content string, err *probe.Error) {
	req, e := http.NewRequest("GET", releaseChecksumURL, nil)
	if e != nil {
		return content, probe.NewError(e)
	}
	req.Header.Set("User-Agent", getUserAgent())

	resp, e := httpClient(timeout).Do(req)
	if e != nil {
		return content, probe.NewError(e)
	}
	if resp == nil {
		return content, probe.NewError(fmt.Errorf("No response from server to download URL %s", releaseChecksumURL))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return content, probe.NewError(fmt.Errorf("Error downloading URL %s. Response: %v", releaseChecksumURL, resp.Status))
	}
	contentBytes, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		return content, probe.NewError(fmt.Errorf("Error reading response. %s", err))
	}

	return string(contentBytes), nil
}

// DownloadReleaseData - downloads release data from mc official server.
func DownloadReleaseData(timeout time.Duration) (data string, err *probe.Error) {
	releaseURL := mcReleaseInfoURL
	if runtime.GOOS == "windows" {
		releaseURL = mcReleaseWindowsInfoURL
	}
	return func() (data string, err *probe.Error) {
		data, err = downloadReleaseURL(releaseURL, timeout)
		if err == nil {
			return data, nil
		}
		return data, err.Trace(releaseURL)
	}()
}

// parseReleaseData - parses release info file content fetched from
// official mc download server.
//
// The expected format is a single line with two words like:
//
// fbe246edbd382902db9a4035df7dce8cb441357d mc.RELEASE.2016-10-07T01-16-39Z
//
// The second word must be `mc.` appended to a standard release tag.
func parseReleaseData(data string) (sha256Hex string, releaseTime time.Time, err *probe.Error) {
	fields := strings.Fields(data)
	if len(fields) != 2 {
		return sha256Hex, releaseTime, probe.NewError(fmt.Errorf("Unknown release data `%s`", data))
	}

	sha256Hex = fields[0]
	releaseInfo := fields[1]

	fields = strings.SplitN(releaseInfo, ".", 2)
	if len(fields) != 2 {
		return sha256Hex, releaseTime, probe.NewError(fmt.Errorf("Unknown release information `%s`", releaseInfo))
	}
	if fields[0] != "mc" {
		return sha256Hex, releaseTime, probe.NewError(fmt.Errorf("Unknown release `%s`", releaseInfo))
	}

	releaseTime, err = releaseTagToReleaseTime(fields[1])
	if err != nil {
		return sha256Hex, releaseTime, err.Trace(fields...)
	}

	return sha256Hex, releaseTime, nil
}

func getLatestReleaseTime(timeout time.Duration) (sha256Hex string, releaseTime time.Time, err *probe.Error) {
	data, err := DownloadReleaseData(timeout)
	if err != nil {
		return sha256Hex, releaseTime, err.Trace()
	}

	return parseReleaseData(data)
}

func getDownloadURL(releaseTag string) (downloadURL string) {
	// Check if we are docker environment, return docker update command
	if IsDocker() {
		// Construct release tag name.
		return fmt.Sprintf("docker pull minio/mc:%s", releaseTag)
	}

	// For binary only installations, we return link to the latest binary.
	if runtime.GOOS == "windows" {
		return mcReleaseURL + "mc.exe"
	}

	return mcReleaseURL + "mc"
}

func getUpdateInfo(timeout time.Duration) (updateMsg string, sha256Hex string, currentReleaseTime, latestReleaseTime time.Time, err *probe.Error) {
	currentReleaseTime, err = GetCurrentReleaseTime()
	if err != nil {
		return updateMsg, sha256Hex, currentReleaseTime, latestReleaseTime, err.Trace()
	}

	sha256Hex, latestReleaseTime, err = getLatestReleaseTime(timeout)
	if err != nil {
		return updateMsg, sha256Hex, currentReleaseTime, latestReleaseTime, err.Trace()
	}

	var older time.Duration
	var downloadURL string
	if latestReleaseTime.After(currentReleaseTime) {
		older = latestReleaseTime.Sub(currentReleaseTime)
		downloadURL = getDownloadURL(releaseTimeToReleaseTag(latestReleaseTime))
	}

	return prepareUpdateMessage(downloadURL, older), sha256Hex, currentReleaseTime, latestReleaseTime, nil
}

var (
	// Check if we stderr, stdout are dumb terminals, we do not apply
	// ansi coloring on dumb terminals.
	isTerminal = func() bool {
		return isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stderr.Fd())
	}

	colorCyanBold = func() func(a ...interface{}) string {
		if isTerminal() {
			color.New(color.FgCyan, color.Bold).SprintFunc()
		}
		return fmt.Sprint
	}()

	colorYellowBold = func() func(format string, a ...interface{}) string {
		if isTerminal() {
			return color.New(color.FgYellow, color.Bold).SprintfFunc()
		}
		return fmt.Sprintf
	}()

	colorGreenBold = func() func(format string, a ...interface{}) string {
		if isTerminal() {
			return color.New(color.FgGreen, color.Bold).SprintfFunc()
		}
		return fmt.Sprintf
	}()
)

func doUpdate(sha256Hex string, latestReleaseTime time.Time, ok bool) (updateStatusMsg string, err *probe.Error) {
	if !ok {
		updateStatusMsg = colorGreenBold("mc update to version RELEASE.%s canceled.",
			latestReleaseTime.Format(mcReleaseTagTimeLayout))
		return updateStatusMsg, nil
	}
	var sha256Sum []byte
	var e error
	sha256Sum, e = hex.DecodeString(sha256Hex)
	if e != nil {
		return updateStatusMsg, probe.NewError(e)
	}

	resp, e := http.Get(getDownloadURL(releaseTimeToReleaseTag(latestReleaseTime)))
	if e != nil {
		return updateStatusMsg, probe.NewError(e)
	}
	defer resp.Body.Close()

	// FIXME: add support for gpg verification as well.
	if e = update.Apply(resp.Body,
		update.Options{
			Hash:     crypto.SHA256,
			Checksum: sha256Sum,
		},
	); e != nil {
		return updateStatusMsg, probe.NewError(e)
	}

	return colorGreenBold("mc updated to version RELEASE.%s successfully.",
		latestReleaseTime.Format(mcReleaseTagTimeLayout)), nil
}

type updateMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// String colorized make bucket message.
func (s updateMessage) String() string {
	return s.Message
}

// JSON jsonified make bucket message.
func (s updateMessage) JSON() string {
	s.Status = "success"
	updateJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(updateJSONBytes)
}

func mainUpdate(ctx *cli.Context) {
	if len(ctx.Args()) != 0 {
		cli.ShowCommandHelpAndExit(ctx, "update", -1)
	}

	globalQuiet = ctx.Bool("quiet") || ctx.GlobalBool("quiet")
	globalJSON = ctx.Bool("json") || ctx.GlobalBool("json")

	updateMsg, sha256Hex, _, latestReleaseTime, err := getUpdateInfo(10 * time.Second)
	if err != nil {
		errorIf(err, "Unable to update ‘mc’.")
		os.Exit(-1)
	}

	// Nothing to update running the latest release.
	color.New(color.FgGreen, color.Bold)
	if updateMsg == "" {
		printMsg(updateMessage{
			Status:  "success",
			Message: colorGreenBold("You are already running the most recent version of ‘mc’."),
		})
		os.Exit(0)
	}

	printMsg(updateMessage{
		Status:  "success",
		Message: updateMsg,
	})

	// Avoid updating mc development, source builds.
	if strings.Contains(updateMsg, mcReleaseURL) {
		var updateStatusMsg string
		var err *probe.Error
		updateStatusMsg, err = doUpdate(sha256Hex, latestReleaseTime, true)
		if err != nil {
			errorIf(err, "Unable to update ‘mc’.")
			os.Exit(-1)
		}
		printMsg(updateMessage{Status: "success", Message: updateStatusMsg})
		os.Exit(1)
	}
}
