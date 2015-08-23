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
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio/pkg/probe"
)

func migrateSession() {
	// Migrate session V1 to V2
	migrateSessionV1ToV2()
}

func createSessionDir() *probe.Error {
	sessionDir, err := getSessionDir()
	if err != nil {
		return err.Trace()
	}

	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return probe.NewError(err)
	}
	return nil
}

func getSessionDir() (string, *probe.Error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}

	sessionDir := filepath.Join(configDir, globalSessionDir)
	return sessionDir, nil
}

func isSessionDirExists() bool {
	sessionDir, err := getSessionDir()
	fatalIf(err.Trace(), "Unable to determine session folder.")

	if _, e := os.Stat(sessionDir); e != nil {
		return false
	}
	return true
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// newSID generates a random session id of regular lower case and uppercase english characters
func newSID(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	sid := make([]rune, n)
	for i := range sid {
		sid[i] = letters[rand.Intn(len(letters))]
	}
	return string(sid)
}

func getSessionFile(sid string) (string, *probe.Error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err.Trace()
	}

	sessionFile := filepath.Join(sessionDir, sid+".json")
	return sessionFile, nil
}

func isSession(sid string) bool {
	sessionFile, err := getSessionFile(sid)
	fatalIf(err.Trace(sid), "Unable to determine session filename for ‘"+sid+"’.")

	if _, e := os.Stat(sessionFile); e != nil {
		return false
	}

	return true // Session exists.
}

func getSessionDataFile(sid string) (string, *probe.Error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err.Trace()
	}

	sessionDataFile := filepath.Join(sessionDir, sid+".data")
	return sessionDataFile, nil
}

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
