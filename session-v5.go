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

func migrateSessionV4ToV5() {
	for _, sid := range getSessionIDs() {
		sessionV5, err := loadSessionV5(sid)
		fatalIf(err.Trace(sid), "Unable to load version ‘5’. Migration failed please report this issue at https://github.com/minio/mc/issues.")
		if sessionV5.Header.Version == "5" { // It is new format.
			return
		}
		/*** Remove all session files older than v5 ***/

		sessionFile, err := getSessionFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session file.")

		sessionDataFile, err := getSessionDataFile(sid)
		fatalIf(err.Trace(sid), "Unable to get session data file.")

		console.Println("Removing unsupported session file ‘" + sessionFile + "’ version ‘" + sessionV5.Header.Version + "’.")
		if e := os.Remove(sessionFile); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove version ‘"+sessionV5.Header.Version+"’ session file ‘"+sessionFile+"’.")
		}
		if e := os.Remove(sessionDataFile); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove version ‘"+sessionV5.Header.Version+"’ session data file ‘"+sessionDataFile+"’.")
		}
	}
}

// sessionV5Header for resumable sessions.
type sessionV5Header struct {
	Version            string            `json:"version"`
	When               time.Time         `json:"time"`
	RootPath           string            `json:"workingFolder"`
	GlobalBoolFlags    map[string]bool   `json:"globalBoolFlags"`
	GlobalIntFlags     map[string]int    `json:"globalIntFlags"`
	GlobalStringFlags  map[string]string `json:"globalStringFlags"`
	CommandType        string            `json:"commandType"`
	CommandArgs        []string          `json:"cmdArgs"`
	CommandBoolFlags   map[string]bool   `json:"cmdBoolFlags"`
	CommandIntFlags    map[string]int    `json:"cmdIntFlags"`
	CommandStringFlags map[string]string `json:"cmdStringFlags"`
	LastCopied         string            `json:"lastCopied"`
	TotalBytes         int64             `json:"totalBytes"`
	TotalObjects       int               `json:"totalObjects"`
}

// sessionMessage container for session messages
type sessionMessage struct {
	Status      string    `json:"status"`
	SessionID   string    `json:"sessionId"`
	Time        time.Time `json:"time"`
	CommandType string    `json:"commandType"`
	CommandArgs []string  `json:"commandArgs"`
}

// sessionV5 resumable session container.
type sessionV5 struct {
	Header    *sessionV5Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *sessionDataFP
	sigCh     bool
}

// sessionDataFP data file pointer.
type sessionDataFP struct {
	dirty bool
	*os.File
}

func (file *sessionDataFP) Write(p []byte) (int, error) {
	file.dirty = true
	return file.File.Write(p)
}

// String colorized session message.
func (s sessionV5) String() string {
	message := console.Colorize("SessionID", fmt.Sprintf("%s -> ", s.SessionID))
	message = message + console.Colorize("SessionTime", fmt.Sprintf("[%s]", s.Header.When.Local().Format(printDate)))
	message = message + console.Colorize("Command", fmt.Sprintf(" %s %s", s.Header.CommandType, strings.Join(s.Header.CommandArgs, " ")))
	return message
}

// JSON jsonified session message.
func (s sessionV5) JSON() string {
	sessionMsg := sessionMessage{
		SessionID:   s.SessionID,
		Time:        s.Header.When.Local(),
		CommandType: s.Header.CommandType,
		CommandArgs: s.Header.CommandArgs,
	}
	sessionMsg.Status = "success"
	sessionBytes, e := json.Marshal(sessionMsg)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(sessionBytes)
}

// newSessionV5 provides a new session.
func newSessionV5() *sessionV5 {
	s := &sessionV5{}
	s.Header = &sessionV5Header{}
	s.Header.Version = "5"
	// map of command and files copied.
	s.Header.GlobalBoolFlags = make(map[string]bool)
	s.Header.GlobalIntFlags = make(map[string]int)
	s.Header.GlobalStringFlags = make(map[string]string)
	s.Header.CommandArgs = nil
	s.Header.CommandBoolFlags = make(map[string]bool)
	s.Header.CommandIntFlags = make(map[string]int)
	s.Header.CommandStringFlags = make(map[string]string)
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
func (s sessionV5) HasData() bool {
	if s.Header.LastCopied == "" {
		return false
	}
	return true
}

// NewDataReader provides reader interface to session data file.
func (s *sessionV5) NewDataReader() io.Reader {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, os.SEEK_SET)
	return io.Reader(s.DataFP)
}

// NewDataReader provides writer interface to session data file.
func (s *sessionV5) NewDataWriter() io.Writer {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, os.SEEK_SET)
	return io.Writer(s.DataFP)
}

// Save this session.
func (s *sessionV5) Save() *probe.Error {
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

// RestoreGlobals restores the state of global variables.
func (s *sessionV5) RestoreGlobals() {
	globalQuietFlag = s.Header.GlobalBoolFlags["quiet"]
	globalJSONFlag = s.Header.GlobalBoolFlags["json"]
	globalDebugFlag = s.Header.GlobalBoolFlags["debug"]
}

// Close ends this session and removes all associated session files.
func (s *sessionV5) Close() *probe.Error {
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
func (s *sessionV5) Delete() *probe.Error {
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
func (s sessionV5) CloseAndDie() {
	s.Close()
	console.Infoln("Session safely terminated. To resume session ‘mc session resume " + s.SessionID + "’")
	os.Exit(0)
}

// loadSessionV5 - reads session file if exists and re-initiates internal variables
func loadSessionV5(sid string) (*sessionV5, *probe.Error) {
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

	s := &sessionV5{}
	s.Header = &sessionV5Header{}
	s.SessionID = sid
	s.Header.Version = "5"
	qs, err := quick.New(s.Header)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}

	s.mutex = new(sync.Mutex)
	s.Header = qs.Data().(*sessionV5Header)

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
