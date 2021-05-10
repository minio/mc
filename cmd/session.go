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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/probe"
)

// migrateSession migrates all previous migration to latest.
func migrateSession() {
	// We no longer support sessions older than v5. They will be removed.
	migrateSessionV5ToV6()

	// Migrate V6 to V7.
	migrateSessionV6ToV7()

	// Migrate V7 to V8
	migrateSessionV7ToV8()
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
	fatalIf(err.Trace(sid), "Unable to determine session filename for `"+sid+"`.")

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
	fatalIf(probe.NewError(e), "Unable to access session folder `"+sessionDir+"`.")

	for _, path := range sessionList {
		sids = append(sids, strings.TrimSuffix(filepath.Base(path), ".json"))
	}
	return sids
}

func getHash(prefix string, args []string) string {
	hasher := sha256.New()
	for _, arg := range args {
		if _, err := hasher.Write([]byte(arg)); err != nil {
			panic(err)
		}
	}

	return prefix + "-" + hex.EncodeToString(hasher.Sum(nil))
}
