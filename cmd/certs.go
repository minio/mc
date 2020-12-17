/*
 * MinIO Client (C) 2016 MinIO, Inc.
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
	"os"
	"path/filepath"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/certs"
)

// getCertsDir - return the full path of certs dir
func getCertsDir() (string, *probe.Error) {
	p, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}
	return filepath.Join(p, globalMCCertsDir), nil
}

// isCertsDirExists - verify if certs directory exists.
func isCertsDirExists() bool {
	certsDir, err := getCertsDir()
	fatalIf(err.Trace(), "Unable to determine certs folder.")
	if _, e := os.Stat(certsDir); e != nil {
		return false
	}
	return true
}

// createCertsDir - create MinIO Client certs folder
func createCertsDir() *probe.Error {
	p, err := getCertsDir()
	if err != nil {
		return err.Trace()
	}
	if e := os.MkdirAll(p, 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// getCAsDir - return the full path of CAs dir
func getCAsDir() (string, *probe.Error) {
	p, err := getCertsDir()
	if err != nil {
		return "", err.Trace()
	}
	return filepath.Join(p, globalMCCAsDir), nil
}

// mustGetCAsDir - return the full path of CAs dir or empty string when an error occurs
func mustGetCAsDir() string {
	p, err := getCAsDir()
	if err != nil {
		return ""
	}
	return p
}

// isCAsDirExists - verify if CAs directory exists.
func isCAsDirExists() bool {
	CAsDir, err := getCAsDir()
	fatalIf(err.Trace(), "Unable to determine CAs folder.")
	if _, e := os.Stat(CAsDir); e != nil {
		return false
	}
	return true
}

// createCAsDir - create MinIO Client CAs folder
func createCAsDir() *probe.Error {
	p, err := getCAsDir()
	if err != nil {
		return err.Trace()
	}
	if e := os.MkdirAll(p, 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// loadRootCAs fetches CA files provided in MinIO config and adds them to globalRootCAs
// Currently under Windows, there is no way to load system + user CAs at the same time
func loadRootCAs() {
	var e error
	globalRootCAs, e = certs.GetRootCAs(mustGetCAsDir())
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to load certificates.")
	}
}
