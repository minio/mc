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
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
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

// SessionMessage container for session messages
type SessionMessage struct {
	SessionID   string    `json:"sessionid"`
	Time        time.Time `json:"time"`
	CommandType string    `json:"command-type"`
	CommandArgs []string  `json:"command-args"`
}

// sessionV2
type sessionV2 struct {
	Header    *sessionV2Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *os.File
	sigCh     bool
}

// String colorized session message
func (s sessionV2) String() string {
	message := console.Colorize("SessionID", fmt.Sprintf("%s -> ", s.SessionID))
	message = message + console.Colorize("SessionTime", fmt.Sprintf("[%s]", s.Header.When.Local().Format(printDate)))
	message = message + console.Colorize("Command", fmt.Sprintf(" %s %s", s.Header.CommandType, strings.Join(s.Header.CommandArgs, " ")))
	return message
}

// JSON jsonified session message
func (s sessionV2) JSON() string {
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

// newSessionV2 provides a new session
func newSessionV2() *sessionV2 {
	s := &sessionV2{}
	s.Header = &sessionV2Header{}
	s.Header.Version = "1.1.0"
	// map of command and files copied
	s.Header.CommandArgs = nil
	s.Header.When = time.Now().UTC()
	s.mutex = new(sync.Mutex)
	s.SessionID = newSID(8)

	sessionDataFile, perr := getSessionDataFile(s.SessionID)
	fatalIf(perr.Trace(s.SessionID), "Unable to create session data file \""+sessionDataFile+"\".")

	var err error
	s.DataFP, err = os.Create(sessionDataFile)
	fatalIf(probe.NewError(err), "Unable to create session data file \""+sessionDataFile+"\".")

	return s
}

// String printer for SessionV2
func (s sessionV2) Info() {
	console.Infoln("Session terminated. To resume session ‘mc session resume " + s.SessionID + "’")
}

// HasData provides true if this is a session resume, false otherwise.
func (s sessionV2) HasData() bool {
	if s.Header.LastCopied == "" {
		return false
	}
	return true
}

// NewDataReader provides reader interface to session data file.
func (s *sessionV2) NewDataReader() io.Reader {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, os.SEEK_SET)
	return io.Reader(s.DataFP)
}

// NewDataReader provides writer interface to session data file.
func (s *sessionV2) NewDataWriter() io.Writer {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, os.SEEK_SET)
	return io.Writer(s.DataFP)
}

// Save this session
func (s *sessionV2) Save() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.DataFP.Sync(); err != nil {
		return probe.NewError(err)
	}

	qs, err := quick.New(s.Header)
	if err != nil {
		return err.Trace(s.SessionID)
	}

	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}
	return qs.Save(sessionFile).Trace()
}

// Close ends this session and removes all associated session files.
func (s *sessionV2) Close() *probe.Error {
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
	return qs.Save(sessionFile).Trace()
}

// Delete removes all the session files.
func (s *sessionV2) Delete() *probe.Error {
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

// loadSession - reads session file if exists and re-initiates internal variables
func loadSessionV2(sid string) (*sessionV2, *probe.Error) {
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

	s := &sessionV2{}
	s.Header = &sessionV2Header{}
	s.SessionID = sid
	s.Header.Version = "1.1.0"
	qs, err := quick.New(s.Header)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}

	s.mutex = new(sync.Mutex)
	s.Header = qs.Data().(*sessionV2Header)

	sessionDataFile, err := getSessionDataFile(s.SessionID)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}

	var e error
	s.DataFP, e = os.Open(sessionDataFile)
	fatalIf(probe.NewError(e), "Unable to open session data file ‘"+sessionDataFile+"’.")

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
