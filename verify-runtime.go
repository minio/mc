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

var minGolangRuntimeVersion = "1.5.1"

// following code handles the current Golang release styles, we might have to update them in future
// if golang community divulges from the below formatting style.
const (
	betaRegexp = "beta[0-9]"
	rcRegexp   = "rc[0-9]"
)

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

type version struct {
	major, minor, patch string
}

func (v1 version) String() string {
	return fmt.Sprintf("%s%s%s", v1.major, v1.minor, v1.patch)
}

func (v1 version) Version() int {
	ver, e := strconv.Atoi(v1.String())
	fatalIf(probe.NewError(e), "Unable to convert version string to an integer.")
	return ver
}

func (v1 version) LessThan(v2 version) bool {
	if v1.Version() < v2.Version() {
		return true
	}
	return false
}

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

func checkGolangRuntimeVersion() {
	v1 := newVersion(getNormalizedGolangVersion())
	v2 := newVersion(minGolangRuntimeVersion)
	if v1.LessThan(v2) {
		fatalIf(errDummy().Trace(),
			"Old Golang runtime version ‘"+v1.String()+"’ detected., ‘mc’ requires minimum go1.5.1 or later.")
	}
}

func verifyMCRuntime() {
	checkGolangRuntimeVersion()

	if !isMcConfigExists() {
		err := createMcConfigDir()
		fatalIf(err.Trace(), "Unable to create ‘mc’ config folder.")

		config, err := newConfig()
		fatalIf(err.Trace(), "Unable to initialize newConfig.")

		err = writeConfig(config)
		fatalIf(err.Trace(), "Unable to write newConfig.")

		console.Infoln("Configuration written to [" + mustGetMcConfigPath() + "]. Please update your access credentials.")
	}
	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session dir.")
	}
	if !isSharedURLsDataDirExists() {
		fatalIf(createSharedURLsDataDir().Trace(), "Unable to create shared url datadir.")
	}
}
