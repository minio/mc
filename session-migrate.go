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

package main

import (
	"os"
	"strconv"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

// Migrates session header version '6' to '7'. Only change is
// LastRemoved field which was added in version '7'.
func migrateSessionV6ToV7() {
	for _, sid := range getSessionIDs() {
		sV6Header, err := loadSessionV6Header(sid)
		fatalIf(err.Trace(sid), "Unable to load version ‘6’. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		if sV6Header.Version == "7" { // It is new format.
			continue
		}

		sessionFile, err := getSessionFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session file.")

		// Initialize v7 header and migrate to new config.
		sV7Header := &sessionV7Header{}
		sV7Header.Version = "7"
		sV7Header.When = sV6Header.When
		sV7Header.RootPath = sV6Header.RootPath
		sV7Header.GlobalBoolFlags = sV6Header.GlobalBoolFlags
		sV7Header.GlobalIntFlags = sV6Header.GlobalIntFlags
		sV7Header.GlobalStringFlags = sV6Header.GlobalStringFlags
		sV7Header.CommandType = sV6Header.CommandType
		sV7Header.CommandArgs = sV6Header.CommandArgs
		sV7Header.CommandBoolFlags = sV6Header.CommandBoolFlags
		sV7Header.CommandIntFlags = sV6Header.CommandIntFlags
		sV7Header.CommandStringFlags = sV6Header.CommandStringFlags
		sV7Header.LastCopied = sV6Header.LastCopied
		sV7Header.LastRemoved = ""
		sV7Header.TotalBytes = sV6Header.TotalBytes
		sV7Header.TotalObjects = sV6Header.TotalObjects

		qs, err := quick.New(sV7Header)
		fatalIf(err.Trace(sid), "Unable to initialize quick config for session '7' header.")

		err = qs.Save(sessionFile)
		fatalIf(err.Trace(sid, sessionFile), "Unable to migrate session from '6' to '7'.")

		console.Println("Successfully migrated ‘" + sessionFile + "’ from version ‘" + sV6Header.Version + "’ to " + "‘" + sV7Header.Version + "’.")
	}
}

// Migrate session version '5' to version '6', all older sessions are
// in-fact removed and not migrated. All session files from '6' and
// above should be migrated - See: migrateSessionV6ToV7().
func migrateSessionV5ToV6() {
	for _, sid := range getSessionIDs() {
		sV6Header, err := loadSessionV6Header(sid)
		fatalIf(err.Trace(sid), "Unable to load version ‘6’. Migration failed please report this issue at https://github.com/minio/mc/issues.")

		sessionVersion, e := strconv.Atoi(sV6Header.Version)
		fatalIf(probe.NewError(e), "Unable to load version ‘6’. Migration failed please report this issue at https://github.com/minio/mc/issues.")

		if sessionVersion > 5 { // It is new format.
			continue
		}

		/*** Remove all session files older than v6 ***/

		sessionFile, err := getSessionFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session file.")

		sessionDataFile, err := getSessionDataFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session data file.")

		console.Println("Removing unsupported session file ‘" + sessionFile + "’ version ‘" + sV6Header.Version + "’.")
		if e := os.Remove(sessionFile); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove version ‘"+sV6Header.Version+"’ session file ‘"+sessionFile+"’.")
		}
		if e := os.Remove(sessionDataFile); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove version ‘"+sV6Header.Version+"’ session data file ‘"+sessionDataFile+"’.")
		}
	}
}
