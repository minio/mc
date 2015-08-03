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

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

func migrateSession() {
	// Migrate session V1 to V2
	migrateSessionV1ToV2()
}

func isSessionDirExists() bool {
	_, err := os.Stat(getSessionDir())
	if err != nil {
		return false
	}
	return true
}

func createSessionDir() *probe.Error {
	if err := os.MkdirAll(getSessionDir(), 0700); err != nil {
		return probe.New(err)
	}
	return nil
}

func getSessionDir() string {
	configDir, err := getMcConfigDir()
	if err != nil {
		// TODO: revamp error handling -ab. Do not pass errors mindlessly to upper layer for a tool like mc.
	}
	return filepath.Join(configDir, sessionDir)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// newSID generates a random session id of regular lower case and uppercase english characters
func newSID(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getSessionFile(sid string) string {
	return filepath.Join(getSessionDir(), sid+".json")
}

func getSessionDataFile(sid string) string {
	return filepath.Join(getSessionDir(), sid+".data")
}

func getSessionIDs() (sids []string) {
	sessionList, err := filepath.Glob(getSessionDir() + "/*.json")
	if err != nil {
		console.Fatalf("Unable to list session folder ‘%s’, %s", getSessionDir(), probe.New(err))
	}

	for _, path := range sessionList {
		sids = append(sids, strings.TrimSuffix(filepath.Base(path), ".json"))
	}
	return sids
}

func isSession(sid string) bool {
	if _, err := os.Stat(getSessionFile(sid)); err != nil {
		return false
	}
	return true
}
