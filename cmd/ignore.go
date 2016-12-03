/*
 * Minio Client (C) 2016 Minio, Inc.
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
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/minio/minio/pkg/probe"
)

// getIgnoreFile - return the full path of ignore file.
func getIgnoreFile() (string, *probe.Error) {
	p, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}
	return filepath.Join(p, globalMCIgnoreFile), nil
}

// mustGetIgnoreFile - return the full path of
func mustGetIgnoreFile() string {
	file, err := getIgnoreFile()
	fatalIf(err.Trace(), "Unable to determine ignore file.")
	return file
}

// isIgnoreFileExists - verify if certs directory exists.
func isIgnoreFileExists() bool {
	ignoreFile, err := getIgnoreFile()
	fatalIf(err.Trace(), "Unable to determine ignore file.")
	if _, e := os.Stat(ignoreFile); e != nil {
		return false
	}
	return true
}

// createIgnoreFile - create minio client ignore file.
func createIgnoreFile() *probe.Error {
	file, err := getIgnoreFile()
	if err != nil {
		return err.Trace()
	}
	if e := ioutil.WriteFile(file, []byte(""), 0600); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Loads the list of ignored files.
func loadIgnoredFiles() {
	ignoreFile := mustGetIgnoreFile()
	file, e := os.Open(ignoreFile)
	fatalIf(probe.NewError(e).Trace(), "Unable to open "+ignoreFile)
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		globalIgnoredFiles = append(globalIgnoredFiles, scan.Text())
	}
	fatalIf(probe.NewError(scan.Err()), "Unable to scan through the "+ignoreFile)
}
