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
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// minGolangRuntimeVersion minimum golang runtime version required for 'mc'.
var minGolangRuntimeVersion = "1.5.1"

// Following code handles the current Golang release styles.
// We might have to update them in future, if golang community
// divulges from the below release styles.
const (
	betaRegexp = "beta[0-9]"
	rcRegexp   = "rc[0-9]"
)

// getNormalizedGolangVersion normalize golang version, handles beta, rc and stable releases.
func getNormalizedGolangVersion() string {
	version := strings.TrimPrefix(runtime.Version(), "go")
	br := regexp.MustCompile(betaRegexp)
	rr := regexp.MustCompile(rcRegexp)
	betaStr := br.FindString(version)
	version = strings.TrimRight(version, betaStr)
	rcStr := rr.FindString(version)
	version = strings.TrimRight(version, rcStr)
	return version
}

// version struct captures the semantic versions.
type version struct {
	major, minor, patch string
}

// String stringify version.
func (v1 version) String() string {
	return fmt.Sprintf("%s%s%s", v1.major, v1.minor, v1.patch)
}

// Version return integer version of String().
func (v1 version) Version() int {
	ver, e := strconv.Atoi(v1.String())
	fatalIf(probe.NewError(e), "Unable to convert version string to an integer.")
	return ver
}

// LessThan figure out if the input version is lesser than current version.
func (v1 version) LessThan(v2 version) bool {
	if v1.Version() < v2.Version() {
		return true
	}
	return false
}

// newVersion convert an input semantic version style string into version struct.
func newVersion(v string) version {
	ver := version{}
	verSlice := strings.Split(v, ".")
	ver.major = verSlice[0]
	ver.minor = verSlice[1]
	if len(verSlice) == 3 {
		ver.patch = verSlice[2]
	} else {
		ver.patch = "0"
	}
	return ver
}

// checkGolangRuntimeVersion - verify currently compiled runtime version.
func checkGolangRuntimeVersion() {
	v1 := newVersion(getNormalizedGolangVersion())
	v2 := newVersion(minGolangRuntimeVersion)
	if v1.LessThan(v2) {
		fatalIf(errDummy().Trace(getNormalizedGolangVersion(), minGolangRuntimeVersion),
			"Old Golang runtime version ‘"+v1.String()+"’ detected., ‘mc’ requires minimum go1.5.1 or later.")
	}
}

// verifyMCRuntime - verify 'mc' compiled runtime version.
func verifyMCRuntime() {
	checkGolangRuntimeVersion()

	// Check if mc config exists.
	if !isMcConfigExists() {
		err := saveMcConfig(newMcConfig())
		fatalIf(err.Trace(), "Unable to save new mc config.")

		console.Infoln("Configuration written to [" + mustGetMcConfigPath() + "]. Please update your access credentials.")
	}

	// Check if mc session folder exists.
	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session config folder.")
	}

	// Check if mc share folder exists.
	if !isShareDirExists() {
		initShareConfig()
	}
}
