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
	"os"
	"strconv"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/minio/pkg/quick"
)

// Migrates session header version '7' to '8'. The only
// change was the adding of insecure global flag
func migrateSessionV7ToV8() {
	for _, sid := range getSessionIDs() {
		sV7, err := loadSessionV7(sid)
		if err != nil {
			if os.IsNotExist(err.ToGoError()) {
				continue
			}
			fatalIf(err.Trace(sid), "Unable to load version `7`. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		}

		// Close underlying session data file.
		sV7.DataFP.Close()

		sessionVersion, e := strconv.Atoi(sV7.Header.Version)
		fatalIf(probe.NewError(e), "Unable to load version `7`. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		if sessionVersion > 7 { // It is new format.
			continue
		}

		sessionFile, err := getSessionFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session file.")

		// Initialize v7 header and migrate to new config.
		sV8Header := &sessionV8Header{}
		sV8Header.Version = globalSessionConfigVersion
		sV8Header.When = sV7.Header.When
		sV8Header.RootPath = sV7.Header.RootPath
		sV8Header.GlobalBoolFlags = sV7.Header.GlobalBoolFlags
		sV8Header.GlobalIntFlags = sV7.Header.GlobalIntFlags
		sV8Header.GlobalStringFlags = sV7.Header.GlobalStringFlags
		sV8Header.CommandType = sV7.Header.CommandType
		sV8Header.CommandArgs = sV7.Header.CommandArgs
		sV8Header.CommandBoolFlags = sV7.Header.CommandBoolFlags
		sV8Header.CommandIntFlags = sV7.Header.CommandIntFlags
		sV8Header.CommandStringFlags = sV7.Header.CommandStringFlags
		sV8Header.LastCopied = sV7.Header.LastCopied
		sV8Header.LastRemoved = sV7.Header.LastRemoved
		sV8Header.TotalBytes = sV7.Header.TotalBytes
		sV8Header.TotalObjects = int64(sV7.Header.TotalObjects)

		// Add insecure flag to the new V8 header
		sV8Header.GlobalBoolFlags["insecure"] = false

		qs, e := quick.NewConfig(sV8Header, nil)
		fatalIf(probe.NewError(e).Trace(sid), "Unable to initialize quick config for session '8' header.")

		e = qs.Save(sessionFile)
		fatalIf(probe.NewError(e).Trace(sid, sessionFile), "Unable to migrate session from '7' to '8'.")

		console.Println("Successfully migrated `" + sessionFile + "` from version `" + sV7.Header.Version + "` to " + "`" + sV8Header.Version + "`.")
	}
}

// Migrates session header version '6' to '7'. Only change is
// LastRemoved field which was added in version '7'.
func migrateSessionV6ToV7() {
	for _, sid := range getSessionIDs() {
		sV6Header, err := loadSessionV6Header(sid)
		if err != nil {
			if os.IsNotExist(err.ToGoError()) {
				continue
			}
			fatalIf(err.Trace(sid), "Unable to load version `6`. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		}

		sessionVersion, e := strconv.Atoi(sV6Header.Version)
		fatalIf(probe.NewError(e), "Unable to load version `6`. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		if sessionVersion > 6 { // It is new format.
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

		qs, e := quick.NewConfig(sV7Header, nil)
		fatalIf(probe.NewError(e).Trace(sid), "Unable to initialize quick config for session '7' header.")

		e = qs.Save(sessionFile)
		fatalIf(probe.NewError(e).Trace(sid, sessionFile), "Unable to migrate session from '6' to '7'.")

		console.Println("Successfully migrated `" + sessionFile + "` from version `" + sV6Header.Version + "` to " + "`" + sV7Header.Version + "`.")
	}
}

// Migrate session version '5' to version '6', all older sessions are
// in-fact removed and not migrated. All session files from '6' and
// above should be migrated - See: migrateSessionV6ToV7().
func migrateSessionV5ToV6() {
	for _, sid := range getSessionIDs() {
		sV6Header, err := loadSessionV6Header(sid)
		if err != nil {
			if os.IsNotExist(err.ToGoError()) {
				continue
			}
			fatalIf(err.Trace(sid), "Unable to load version `6`. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		}

		sessionVersion, e := strconv.Atoi(sV6Header.Version)
		fatalIf(probe.NewError(e), "Unable to load version `6`. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		if sessionVersion > 5 { // It is new format.
			continue
		}

		/*** Remove all session files older than v6 ***/

		sessionFile, err := getSessionFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session file.")

		sessionDataFile, err := getSessionDataFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session data file.")

		console.Println("Removing unsupported session file `" + sessionFile + "` version `" + sV6Header.Version + "`.")
		if e := os.Remove(sessionFile); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove version `"+sV6Header.Version+"` session file `"+sessionFile+"`.")
		}
		if e := os.Remove(sessionDataFile); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove version `"+sV6Header.Version+"` session data file `"+sessionDataFile+"`.")
		}
	}
}
