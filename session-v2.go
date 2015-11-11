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

// Session V2 - Version 2 stores session header and session data in
// two separate files. Session data contains fully prepared URL list.
package main

import (
	"os"
	"time"

	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

// migrateSessionV1ToV2 migrates all session files from v1 to v2.
// This function should be called from the main early on.
func migrateSessionV1ToV2() {
	for _, sid := range getSessionIDsV1() {
		sessionFileV1, err := getSessionFileV1(sid)
		fatalIf(err.Trace(sid), "Unable to determine session file for ‘"+sid+"’.")

		e := os.Remove(sessionFileV1)
		fatalIf(probe.NewError(e), "Migration failed. Unable to remove old session file ‘"+sessionFileV1+"’.")
	}
}

// sessionV2Header
type sessionV2Header struct {
	Version      string    `json:"version"`
	When         time.Time `json:"time"`
	RootPath     string    `json:"working-folder"`
	CommandType  string    `json:"command-type"`
	CommandArgs  []string  `json:"cmd-args"`
	LastCopied   string    `json:"last-copied"`
	TotalBytes   int64     `json:"total-bytes"`
	TotalObjects int       `json:"total-objects"`
}

// loadSession - reads session file if exists and re-initiates internal variables
func loadSessionV2(sid string) (*sessionV2Header, *probe.Error) {
	if !isSessionDirExists() {
		return nil, errInvalidArgument().Trace()
	}
	sessionFile, err := getSessionFile(sid)
	if err != nil {
		return nil, err.Trace(sid)
	}
	if _, err := os.Stat(sessionFile); err != nil {
		return nil, probe.NewError(err)
	}
	sessionHeader := &sessionV2Header{}
	sessionHeader.Version = "1.1.0"
	qs, err := quick.New(sessionHeader)
	if err != nil {
		return nil, err.Trace(sid, sessionHeader.Version)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, err.Trace(sid, sessionHeader.Version)
	}
	return sessionHeader, nil
}
