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
)

var minWindowsGolangVersion = "1.5"
var minGolangVersion = "1.3"

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

func newVersion(v string) version {
	var ver version
	verSlice := strings.Split(v, ".")
	if len(verSlice) > 2 {
		ver = version{
			major: verSlice[0],
			minor: verSlice[1],
			patch: verSlice[2],
		}
		return ver
	}
	ver = version{
		major: verSlice[0],
		minor: verSlice[1],
		patch: "0",
	}
	return ver
}

func (v1 version) String() string {
	return fmt.Sprintf("%s%s%s", v1.major, v1.minor, v1.patch)
}

func (v1 version) Version() int {
	ver, err := strconv.Atoi(v1.String())
	if err != nil {
		console.Fatalf("Unable to parse version string. %s\n", err)
	}
	return ver
}

func (v1 version) LessThan(v2 version) bool {
	if v1.Version() < v2.Version() {
		return true
	}
	return false
}

func checkGolangRuntimeVersion() {
	v1 := newVersion(getNormalizedGolangVersion())
	switch runtime.GOOS {
	case "windows":
		v2 := newVersion(minWindowsGolangVersion)
		if v1.LessThan(v2) {
			console.Errorln("Minimum Golang runtime expected on windows is go1.5, please compile ‘mc’ with atleast go1.5")
		}
	default:
		v2 := newVersion(minGolangVersion)
		if v1.LessThan(v2) {
			console.Errorln("Minimum Golang runtime expected on windows is go1.3, please compile ‘mc’ with atleast go1.3")
		}
	}
}

func verifyMCRuntime() {
	if !isMcConfigExists() {
		if err := createMcConfigDir(); err != nil {
			console.Fatalf("Unable to create ‘mc’ config folder. %s\n", err)
		}
		config, err := newConfig()
		fatalIf(err)
		err = writeConfig(config)
		fatalIf(err)
		console.Infoln("Configuration written to [" + mustGetMcConfigPath() + "]. Please update your access credentials.")
	}
	if !isSessionDirExists() {
		err := createSessionDir()
		fatalIf(err)
	}
	checkGolangRuntimeVersion()
}
