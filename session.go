/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio/pkg/probe"
)

// migrateSession migrates all previous migration to latest.
func migrateSession() {
	// We no longer support sessions older than v5. They will be removed.
	migrateSessionV5ToV6()

	// Migrate V6 to V7.
	migrateSessionV6ToV7()
}

// createSessionDir - create session directory.
func createSessionDir() *probe.Error {
	sessionDir, err := getSessionDir()
	if err != nil {
		return err.Trace()
	}

	if e := os.MkdirAll(sessionDir, 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// getSessionDir - get session directory.
func getSessionDir() (string, *probe.Error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}

	sessionDir := filepath.Join(configDir, globalSessionDir)
	return sessionDir, nil
}

// isSessionDirExists - verify if session directory exists.
func isSessionDirExists() bool {
	sessionDir, err := getSessionDir()
	fatalIf(err.Trace(), "Unable to determine session folder.")

	if _, e := os.Stat(sessionDir); e != nil {
		return false
	}
	return true
}

// getSessionFile - get current session file.
func getSessionFile(sid string) (string, *probe.Error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err.Trace()
	}

	sessionFile := filepath.Join(sessionDir, sid+".json")
	return sessionFile, nil
}

// isSessionExists verifies if given session exists.
func isSessionExists(sid string) bool {
	sessionFile, err := getSessionFile(sid)
	fatalIf(err.Trace(sid), "Unable to determine session filename for ‘"+sid+"’.")

	if _, e := os.Stat(sessionFile); e != nil {
		return false
	}

	return true // Session exists.
}

// getSessionDataFile - get session data file for a given session.
func getSessionDataFile(sid string) (string, *probe.Error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err.Trace()
	}

	sessionDataFile := filepath.Join(sessionDir, sid+".data")
	return sessionDataFile, nil
}

// getSessionIDs - get all active sessions.
func getSessionIDs() (sids []string) {
	sessionDir, err := getSessionDir()
	fatalIf(err.Trace(), "Unable to access session folder.")

	sessionList, e := filepath.Glob(sessionDir + "/*.json")
	fatalIf(probe.NewError(e), "Unable to access session folder ‘"+sessionDir+"’.")

	for _, path := range sessionList {
		sids = append(sids, strings.TrimSuffix(filepath.Base(path), ".json"))
	}
	return sids
}

// removeSessionFile - remove the session file, ending with .json
func removeSessionFile(sid string) {
	sessionFile, err := getSessionFile(sid)
	if err != nil {
		return
	}
	os.Remove(sessionFile)
}

// removeSessionDataFile - remove the session data file, ending with .data
func removeSessionDataFile(sid string) {
	dataFile, err := getSessionDataFile(sid)
	if err != nil {
		return
	}
	os.Remove(dataFile)
}
