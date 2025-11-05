// Copyright (c) 2015-2022 MinIO, Inc.
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
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "crypto/sha256" // needed for selfupdate hashers

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/env"
	"github.com/minio/selfupdate"
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

	envMinisignPubKey = "MC_UPDATE_MINISIGN_PUBKEY"
)

// For windows our files have .exe additionally.
var mcReleaseWindowsInfoURL = mcReleaseURL + "mc.exe.sha256sum"

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

// releaseTagToReleaseTime - releaseTag to releaseTime
func releaseTagToReleaseTime(releaseTag string) (releaseTime time.Time, err *probe.Error) {
	fields := strings.Split(releaseTag, ".")
	if len(fields) < 2 || len(fields) > 4 {
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
//	"/.dockerenv":      "file",
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
//	mc (<OS>; <ARCH>[; dcos][; kubernetes][; docker][; source]) mc/<VERSION> mc/<RELEASE-TAG> mc/<COMMIT-ID>
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
	contentBytes, e := io.ReadAll(resp.Body)
	if e != nil {
		return content, probe.NewError(fmt.Errorf("Error reading response. %s", err))
	}

	return string(contentBytes), nil
}

// DownloadReleaseData - downloads release data from mc official server.
func DownloadReleaseData(customReleaseURL string, timeout time.Duration) (data string, err *probe.Error) {
	releaseURL := mcReleaseInfoURL
	if runtime.GOOS == "windows" {
		releaseURL = mcReleaseWindowsInfoURL
	}
	if customReleaseURL != "" {
		releaseURL = customReleaseURL
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
func parseReleaseData(data string) (sha256Hex string, releaseTime time.Time, releaseTag string, err *probe.Error) {
	fields := strings.Fields(data)
	if len(fields) != 2 {
		return sha256Hex, releaseTime, "", probe.NewError(fmt.Errorf("Unknown release data `%s`", data))
	}

	sha256Hex = fields[0]
	releaseInfo := fields[1]

	fields = strings.SplitN(releaseInfo, ".", 2)
	if len(fields) != 2 {
		return sha256Hex, releaseTime, "", probe.NewError(fmt.Errorf("Unknown release information `%s`", releaseInfo))
	}
	if fields[0] != "mc" {
		return sha256Hex, releaseTime, "", probe.NewError(fmt.Errorf("Unknown release `%s`", releaseInfo))
	}

	releaseTime, err = releaseTagToReleaseTime(fields[1])
	if err != nil {
		return sha256Hex, releaseTime, fields[1], err.Trace(fields...)
	}

	return sha256Hex, releaseTime, fields[1], nil
}

func getLatestReleaseTime(customReleaseURL string, timeout time.Duration) (sha256Hex string, releaseTime time.Time, releaseTag string, err *probe.Error) {
	data, err := DownloadReleaseData(customReleaseURL, timeout)
	if err != nil {
		return sha256Hex, releaseTime, releaseTag, err.Trace()
	}

	return parseReleaseData(data)
}

func getDownloadURL(customReleaseURL, releaseTag string) (downloadURL string) {
	// Check if we are docker environment, return docker update command
	if IsDocker() {
		// Construct release tag name.
		return fmt.Sprintf("docker pull minio/mc:%s", releaseTag)
	}

	if customReleaseURL == "" {
		return mcReleaseURL + "archive/mc." + releaseTag
	}

	u, e := url.Parse(customReleaseURL)
	if e != nil {
		return mcReleaseURL + "archive/mc." + releaseTag
	}

	u.Path = path.Dir(u.Path) + "/mc." + releaseTag
	return u.String()
}

func getUpdateInfo(customReleaseURL string, timeout time.Duration) (updateMsg, sha256Hex string, currentReleaseTime, latestReleaseTime time.Time, releaseTag string, err *probe.Error) {
	currentReleaseTime, err = GetCurrentReleaseTime()
	if err != nil {
		return updateMsg, sha256Hex, currentReleaseTime, latestReleaseTime, releaseTag, err.Trace()
	}

	sha256Hex, latestReleaseTime, releaseTag, err = getLatestReleaseTime(customReleaseURL, timeout)
	if err != nil {
		return updateMsg, sha256Hex, currentReleaseTime, latestReleaseTime, releaseTag, err.Trace()
	}

	var older time.Duration
	var downloadURL string
	if latestReleaseTime.After(currentReleaseTime) {
		older = latestReleaseTime.Sub(currentReleaseTime)
		downloadURL = getDownloadURL(customReleaseURL, releaseTag)
	}

	return prepareUpdateMessage(downloadURL, older), sha256Hex, currentReleaseTime, latestReleaseTime, releaseTag, nil
}

var (
	// Check if we stderr, stdout are dumb terminals, we do not apply
	// ansi coloring on dumb terminals.
	isTerminal = func() bool {
		return isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stderr.Fd())
	}

	colorCyanBold = func() func(a ...any) string {
		if isTerminal() {
			color.New(color.FgCyan, color.Bold).SprintFunc()
		}
		return fmt.Sprint
	}()

	colorYellowBold = func() func(format string, a ...any) string {
		if isTerminal() {
			return color.New(color.FgYellow, color.Bold).SprintfFunc()
		}
		return fmt.Sprintf
	}()

	colorGreenBold = func() func(format string, a ...any) string {
		if isTerminal() {
			return color.New(color.FgGreen, color.Bold).SprintfFunc()
		}
		return fmt.Sprintf
	}()
)

func getUpdateTransport(timeout time.Duration) http.RoundTripper {
	var updateTransport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: timeout,
			DualStack: true,
		}).DialContext,
		IdleConnTimeout:       timeout,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: timeout,
		TLSClientConfig: &tls.Config{
			RootCAs: globalRootCAs,
		},
		DisableCompression: true,
	}
	return updateTransport
}

func getUpdateReaderFromURL(u *url.URL, transport http.RoundTripper) (io.ReadCloser, error) {
	clnt := &http.Client{
		Transport: transport,
	}
	req, e := http.NewRequest(http.MethodGet, u.String(), nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", getUserAgent())

	resp, e := clnt.Do(req)
	if e != nil {
		return nil, e
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	return newProgressReader(resp.Body, "mc", resp.ContentLength), nil
}

func doUpdate(customReleaseURL, sha256Hex string, latestReleaseTime time.Time, releaseTag string, ok bool) (updateStatusMsg string, err *probe.Error) {
	fmtReleaseTime := latestReleaseTime.Format(mcReleaseTagTimeLayout)
	if !ok {
		updateStatusMsg = colorGreenBold("mc update to version %s canceled.",
			releaseTag)
		return updateStatusMsg, nil
	}

	sha256Sum, e := hex.DecodeString(sha256Hex)
	if e != nil {
		return updateStatusMsg, probe.NewError(e)
	}

	u, e := url.Parse(getDownloadURL(customReleaseURL, releaseTag))
	if e != nil {
		return updateStatusMsg, probe.NewError(e)
	}

	transport := getUpdateTransport(30 * time.Second)

	rc, e := getUpdateReaderFromURL(u, transport)
	if e != nil {
		return updateStatusMsg, probe.NewError(e)
	}
	defer rc.Close()

	opts := selfupdate.Options{
		Hash:     crypto.SHA256,
		Checksum: sha256Sum,
	}

	minisignPubkey := env.Get(envMinisignPubKey, "")
	if minisignPubkey != "" {
		v := selfupdate.NewVerifier()
		u.Path = path.Dir(u.Path) + "/mc." + releaseTag + ".minisig"
		if e = v.LoadFromURL(u.String(), minisignPubkey, transport); e != nil {
			return updateStatusMsg, probe.NewError(e)
		}
		opts.Verifier = v
	}

	if e := opts.CheckPermissions(); e != nil {
		permErrMsg := fmt.Sprintf(" failed with: %s", e)
		updateStatusMsg = colorYellowBold("mc update to version RELEASE.%s %s.",
			fmtReleaseTime, permErrMsg)
		return updateStatusMsg, nil
	}

	if e = selfupdate.Apply(rc, opts); e != nil {
		if re := selfupdate.RollbackError(e); re != nil {
			rollBackErr := fmt.Sprintf("Failed to rollback from bad update: %v", re)
			updateStatusMsg = colorYellowBold("mc update to version RELEASE.%s %s.", fmtReleaseTime, rollBackErr)
			return updateStatusMsg, probe.NewError(e)
		}

		var pathErr *os.PathError
		if errors.As(e, &pathErr) {
			pathErrMsg := fmt.Sprintf("Unable to update the binary at %s: %v", filepath.Dir(pathErr.Path), pathErr.Err)
			updateStatusMsg = colorYellowBold("mc update to version RELEASE.%s %s.",
				fmtReleaseTime, pathErrMsg)
			return updateStatusMsg, nil
		}

		return colorYellowBold(fmt.Sprintf("Error in mc update to version RELEASE.%s %v.", fmtReleaseTime, e)), nil
	}

	return colorGreenBold("mc updated to version RELEASE.%s successfully.", fmtReleaseTime), nil
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
	if len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, -1)
	}

	globalQuiet = ctx.Bool("quiet") || ctx.GlobalBool("quiet")
	globalJSON = ctx.Bool("json") || ctx.GlobalBool("json")

	customReleaseURL := ctx.Args().Get(0)

	updateMsg, sha256Hex, _, latestReleaseTime, releaseTag, err := getUpdateInfo(customReleaseURL, 10*time.Second)
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
	if updateMsg != "" {
		var updateStatusMsg string
		var err *probe.Error
		updateStatusMsg, err = doUpdate(customReleaseURL, sha256Hex, latestReleaseTime, releaseTag, true)
		if err != nil {
			errorIf(err, "Unable to update ‘mc’.")
			os.Exit(-1)
		}
		printMsg(updateMessage{Status: "success", Message: updateStatusMsg})
		os.Exit(1)
	}
}
