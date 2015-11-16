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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

func migrateSessionV2ToV3() {
	for _, sid := range getSessionIDs() {
		oldSessionV2Header, err := loadSessionV2(sid)
		// 1.1.0 intermediate version number is actually v2.
		fatalIf(err.Trace(sid), "Unable to load version ‘1.1.0’. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		if oldSessionV2Header.Version == "3" {
			return
		}
		newSessionV3Header := &sessionV3Header{}
		newSessionV3Header.Version = "3"
		newSessionV3Header.When = oldSessionV2Header.When
		newSessionV3Header.RootPath = oldSessionV2Header.RootPath
		newSessionV3Header.CommandType = oldSessionV2Header.CommandType
		newSessionV3Header.CommandArgs = oldSessionV2Header.CommandArgs
		newSessionV3Header.LastCopied = oldSessionV2Header.LastCopied
		newSessionV3Header.TotalBytes = oldSessionV2Header.TotalBytes
		newSessionV3Header.TotalObjects = oldSessionV2Header.TotalObjects

		qs, err := quick.New(newSessionV3Header)
		fatalIf(err.Trace(sid), "Unable to initialize quick config.")

		sessionFile, err := getSessionFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session file.")

		err = qs.Save(sessionFile)
		fatalIf(err.Trace(sid), "Saving new session file for version ‘3’ failed.")
	}
}

// sessionV3Header
type sessionV3Header struct {
	Version         string    `json:"version"`
	When            time.Time `json:"time"`
	RootPath        string    `json:"workingFolder"`
	CommandType     string    `json:"commandType"`
	CommandArgs     []string  `json:"cmdArgs"`
	CommandBoolFlag struct {
		Key   string
		Value bool
	} `json:"cmdBoolFlag"`
	CommandIntFlag struct {
		Key   string
		Value int
	} `json:"cmdIntFlag"`
	CommandStringFlag struct {
		Key   string
		Value string
	} `json:"cmdStringFlag"`
	LastCopied   string `json:"lastCopied"`
	TotalBytes   int64  `json:"totalBytes"`
	TotalObjects int    `json:"totalObjects"`
}

// SessionMessage container for session messages
type SessionMessage struct {
	SessionID   string    `json:"sessionId"`
	Time        time.Time `json:"time"`
	CommandType string    `json:"commandType"`
	CommandArgs []string  `json:"commandArgs"`
}

// sessionV3
type sessionV3 struct {
	Header    *sessionV3Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *sessionDataFP
	sigCh     bool
}

type sessionDataFP struct {
	dirty bool
	*os.File
}

func (file *sessionDataFP) Write(p []byte) (int, error) {
	file.dirty = true
	return file.File.Write(p)
}

// String colorized session message
func (s sessionV3) String() string {
	message := console.Colorize("SessionID", fmt.Sprintf("%s -> ", s.SessionID))
	message = message + console.Colorize("SessionTime", fmt.Sprintf("[%s]", s.Header.When.Local().Format(printDate)))
	message = message + console.Colorize("Command", fmt.Sprintf(" %s %s", s.Header.CommandType, strings.Join(s.Header.CommandArgs, " ")))
	return message
}

// JSON jsonified session message
func (s sessionV3) JSON() string {
	sessionMesage := SessionMessage{
		SessionID:   s.SessionID,
		Time:        s.Header.When.Local(),
		CommandType: s.Header.CommandType,
		CommandArgs: s.Header.CommandArgs,
	}
	sessionBytes, e := json.Marshal(sessionMesage)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(sessionBytes)
}

// newSessionV3 provides a new session
func newSessionV3() *sessionV3 {
	s := &sessionV3{}
	s.Header = &sessionV3Header{}
	s.Header.Version = "3"
	// map of command and files copied
	s.Header.CommandArgs = nil
	s.Header.CommandBoolFlag.Key = ""
	s.Header.CommandBoolFlag.Value = false
	s.Header.CommandIntFlag.Key = ""
	s.Header.CommandIntFlag.Value = 0
	s.Header.CommandStringFlag.Key = ""
	s.Header.CommandStringFlag.Value = ""
	s.Header.When = time.Now().UTC()
	s.mutex = new(sync.Mutex)
	s.SessionID = newRandomID(8)

	sessionDataFile, err := getSessionDataFile(s.SessionID)
	fatalIf(err.Trace(s.SessionID), "Unable to create session data file \""+sessionDataFile+"\".")

	dataFile, e := os.Create(sessionDataFile)
	fatalIf(probe.NewError(e), "Unable to create session data file \""+sessionDataFile+"\".")

	s.DataFP = &sessionDataFP{false, dataFile}
	return s
}

// HasData provides true if this is a session resume, false otherwise.
func (s sessionV3) HasData() bool {
	if s.Header.LastCopied == "" {
		return false
	}
	return true
}

// NewDataReader provides reader interface to session data file.
func (s *sessionV3) NewDataReader() io.Reader {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, os.SEEK_SET)
	return io.Reader(s.DataFP)
}

// NewDataReader provides writer interface to session data file.
func (s *sessionV3) NewDataWriter() io.Writer {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, os.SEEK_SET)
	return io.Writer(s.DataFP)
}

// Save this session
func (s *sessionV3) Save() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.DataFP.dirty {
		if err := s.DataFP.Sync(); err != nil {
			return probe.NewError(err)
		}
		s.DataFP.dirty = false
	}

	qs, err := quick.New(s.Header)
	if err != nil {
		return err.Trace(s.SessionID)
	}

	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}
	return qs.Save(sessionFile).Trace(sessionFile)
}

// Close ends this session and removes all associated session files.
func (s *sessionV3) Close() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.DataFP.Close(); err != nil {
		return probe.NewError(err)
	}

	qs, err := quick.New(s.Header)
	if err != nil {
		return err.Trace()
	}

	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}
	return qs.Save(sessionFile).Trace(sessionFile)
}

// Delete removes all the session files.
func (s *sessionV3) Delete() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.DataFP != nil {
		name := s.DataFP.Name()
		// close file pro-actively before deleting
		// ignore any error, it could be possibly that
		// the file is closed already
		s.DataFP.Close()

		err := os.Remove(name)
		if err != nil {
			return probe.NewError(err)
		}
	}

	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}

	if err := os.Remove(sessionFile); err != nil {
		return probe.NewError(err)
	}

	return nil
}

// Close a session and exit.
func (s sessionV3) CloseAndDie() {
	s.Close()
	console.Infoln("Session safely terminated. To resume session ‘mc session resume " + s.SessionID + "’")
	os.Exit(0)
}

// loadSessionV3 - reads session file if exists and re-initiates internal variables
func loadSessionV3(sid string) (*sessionV3, *probe.Error) {
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

	s := &sessionV3{}
	s.Header = &sessionV3Header{}
	s.SessionID = sid
	s.Header.Version = "3"
	qs, err := quick.New(s.Header)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}

	s.mutex = new(sync.Mutex)
	s.Header = qs.Data().(*sessionV3Header)

	sessionDataFile, err := getSessionDataFile(s.SessionID)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}

	var e error
	dataFile, e := os.Open(sessionDataFile)
	fatalIf(probe.NewError(e), "Unable to open session data file \""+sessionDataFile+"\".")

	s.DataFP = &sessionDataFP{false, dataFile}

	return s, nil
}

// Create a factory function to simplify checking if an
// object has been copied or not.
// isCopied(URL) -> true or false
func isCopiedFactory(lastCopied string) func(string) bool {
	copied := true // closure
	return func(sourceURL string) bool {
		if sourceURL == "" {
			fatalIf(errInvalidArgument().Trace(), "Empty source argument passed.")
		}
		if lastCopied == "" {
			return false
		}

		if copied {
			if lastCopied == sourceURL {
				copied = false // from next call onwards we say false.
			}
			return true
		}
		return false
	}
}
