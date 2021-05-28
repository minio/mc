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

// Package cmd - session V8 - Version 8 stores session header and session data in
// two separate files. Session data contains fully prepared URL list.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/minio/pkg/quick"
)

// sessionV8Header for resumable sessions.
type sessionV8Header struct {
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
	LastRemoved        string            `json:"lastRemoved"`
	TotalBytes         int64             `json:"totalBytes"`
	TotalObjects       int64             `json:"totalObjects"`
	UserMetaData       map[string]string `json:"metaData"`
}

// sessionMessage container for session messages
type sessionMessage struct {
	Status      string    `json:"status"`
	SessionID   string    `json:"sessionId"`
	Time        time.Time `json:"time"`
	CommandType string    `json:"commandType"`
	CommandArgs []string  `json:"commandArgs"`
}

// sessionV8 resumable session container.
type sessionV8 struct {
	Header    *sessionV8Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *sessionDataFP
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
func (s sessionV8) String() string {
	message := console.Colorize("SessionID", fmt.Sprintf("%s -> ", s.SessionID))
	message = message + console.Colorize("SessionTime", fmt.Sprintf("[%s]", s.Header.When.Local().Format(printDate)))
	message = message + console.Colorize("Command", fmt.Sprintf(" %s %s", s.Header.CommandType, strings.Join(s.Header.CommandArgs, " ")))
	return message
}

// JSON jsonified session message.
func (s sessionV8) JSON() string {
	sessionMsg := sessionMessage{
		SessionID:   s.SessionID,
		Time:        s.Header.When.Local(),
		CommandType: s.Header.CommandType,
		CommandArgs: s.Header.CommandArgs,
	}
	sessionMsg.Status = "success"
	sessionBytes, e := json.MarshalIndent(sessionMsg, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(sessionBytes)
}

// loadSessionV8 - reads session file if exists and re-initiates internal variables
func loadSessionV8(sid string) (*sessionV8, *probe.Error) {
	if !isSessionDirExists() {
		return nil, errInvalidArgument().Trace()
	}
	sessionFile, err := getSessionFile(sid)
	if err != nil {
		return nil, err.Trace(sid)
	}

	if _, e := os.Stat(sessionFile); e != nil {
		return nil, probe.NewError(e)
	}

	// Initialize new session.
	s := &sessionV8{
		Header: &sessionV8Header{
			Version: globalSessionConfigVersion,
		},
		SessionID: sid,
	}

	// Initialize session config loader.
	qs, e := quick.NewConfig(s.Header, nil)
	if e != nil {
		return nil, probe.NewError(e).Trace(sid, s.Header.Version)
	}

	if e = qs.Load(sessionFile); e != nil {
		return nil, probe.NewError(e).Trace(sid, s.Header.Version)
	}

	// Validate if the version matches with expected current version.
	sV8Header := qs.Data().(*sessionV8Header)
	if sV8Header.Version != globalSessionConfigVersion {
		msg := fmt.Sprintf("Session header version %s does not match mc session version %s.\n",
			sV8Header.Version, globalSessionConfigVersion)
		return nil, probe.NewError(errors.New(msg)).Trace(sid, sV8Header.Version)
	}

	s.mutex = new(sync.Mutex)
	s.Header = sV8Header

	sessionDataFile, err := getSessionDataFile(s.SessionID)
	if err != nil {
		return nil, err.Trace(sid, s.Header.Version)
	}

	dataFile, e := os.Open(sessionDataFile)
	if e != nil {
		return nil, probe.NewError(e)
	}
	s.DataFP = &sessionDataFP{false, dataFile}

	return s, nil
}

// newSessionV8 provides a new session.
func newSessionV8(sessionID string) *sessionV8 {
	s := &sessionV8{}
	s.Header = &sessionV8Header{}
	s.Header.Version = globalSessionConfigVersion
	// map of command and files copied.
	s.Header.GlobalBoolFlags = make(map[string]bool)
	s.Header.GlobalIntFlags = make(map[string]int)
	s.Header.GlobalStringFlags = make(map[string]string)
	s.Header.CommandArgs = nil
	s.Header.CommandBoolFlags = make(map[string]bool)
	s.Header.CommandIntFlags = make(map[string]int)
	s.Header.CommandStringFlags = make(map[string]string)
	s.Header.UserMetaData = make(map[string]string)
	s.Header.When = UTCNow()
	s.mutex = new(sync.Mutex)
	s.SessionID = sessionID

	sessionDataFile, err := getSessionDataFile(s.SessionID)
	fatalIf(err.Trace(s.SessionID), "Unable to create session data file \""+sessionDataFile+"\".")

	dataFile, e := os.Create(sessionDataFile)
	fatalIf(probe.NewError(e), "Unable to create session data file \""+sessionDataFile+"\".")

	s.DataFP = &sessionDataFP{false, dataFile}

	// Capture state of global flags.
	s.setGlobals()

	return s
}

// HasData provides true if this is a session resume, false otherwise.
func (s sessionV8) HasData() bool {
	return s.Header.LastCopied != "" || s.Header.LastRemoved != ""
}

// NewDataReader provides reader interface to session data file.
func (s *sessionV8) NewDataReader() io.Reader {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, io.SeekStart)
	return io.Reader(s.DataFP)
}

// NewDataReader provides writer interface to session data file.
func (s *sessionV8) NewDataWriter() io.Writer {
	// DataFP is always intitialized, either via new or load functions.
	s.DataFP.Seek(0, io.SeekStart)
	// when moving to file position 0 we want to truncate the file as well,
	// otherwise we'll partly overwrite existing data
	s.DataFP.Truncate(0)
	return io.Writer(s.DataFP)
}

// Save this session.
func (s *sessionV8) Save() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.DataFP.dirty {
		if err := s.DataFP.Sync(); err != nil {
			return probe.NewError(err)
		}
		s.DataFP.dirty = false
	}

	qs, e := quick.NewConfig(s.Header, nil)
	if e != nil {
		return probe.NewError(e).Trace(s.SessionID)
	}

	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}
	e = qs.Save(sessionFile)
	if e != nil {
		return probe.NewError(e).Trace(sessionFile)
	}
	return nil
}

// setGlobals captures the state of global variables into session header.
// Used by newSession.
func (s *sessionV8) setGlobals() {
	s.Header.GlobalBoolFlags["quiet"] = globalQuiet
	s.Header.GlobalBoolFlags["debug"] = globalDebug
	s.Header.GlobalBoolFlags["json"] = globalJSON
	s.Header.GlobalBoolFlags["noColor"] = globalNoColor
	s.Header.GlobalBoolFlags["insecure"] = globalInsecure
}

// IsModified - returns if in memory session header has changed from
// its on disk value.
func (s *sessionV8) isModified(sessionFile string) (bool, *probe.Error) {
	qs, e := quick.NewConfig(s.Header, nil)
	if e != nil {
		return false, probe.NewError(e).Trace(s.SessionID)
	}

	var currentHeader = &sessionV8Header{}
	currentQS, e := quick.LoadConfig(sessionFile, nil, currentHeader)
	if e != nil {
		// If session does not exist for the first, return modified to
		// be true.
		if os.IsNotExist(e) {
			return true, nil
		}
		// For all other errors return.
		return false, probe.NewError(e).Trace(s.SessionID)
	}

	changedFields, e := qs.DeepDiff(currentQS)
	if e != nil {
		return false, probe.NewError(e).Trace(s.SessionID)
	}

	// Returns true if there are changed entries.
	return len(changedFields) > 0, nil
}

// save - wrapper for quick.Save and saves only if sessionHeader is
// modified.
func (s *sessionV8) save() *probe.Error {
	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}

	// Verify if sessionFile is modified.
	modified, err := s.isModified(sessionFile)
	if err != nil {
		return err.Trace(s.SessionID)
	}
	// Header is modified, we save it.
	if modified {
		qs, e := quick.NewConfig(s.Header, nil)
		if e != nil {
			return probe.NewError(e).Trace(s.SessionID)
		}
		// Save an return.
		e = qs.Save(sessionFile)
		if e != nil {
			return probe.NewError(e).Trace(sessionFile)
		}
	}
	return nil
}

// Close ends this session and removes all associated session files.
func (s *sessionV8) Close() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.DataFP.Close(); err != nil {
		return probe.NewError(err)
	}

	// Attempt to save the header if modified.
	return s.save()
}

// Delete removes all the session files.
func (s *sessionV8) Delete() *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.DataFP != nil {
		name := s.DataFP.Name()
		// close file pro-actively before deleting
		// ignore any error, it could be possibly that
		// the file is closed already
		s.DataFP.Close()

		// Remove the data file.
		if e := os.Remove(name); e != nil {
			return probe.NewError(e)
		}
	}

	// Fetch the session file.
	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err.Trace(s.SessionID)
	}

	// Remove session file
	if e := os.Remove(sessionFile); e != nil {
		return probe.NewError(e)
	}

	// Remove session backup file if any, ignore any error.
	os.Remove(sessionFile + ".old")

	return nil
}

// Close a session and exit.
func (s sessionV8) CloseAndDie() {
	s.Close()
	console.Fatalln("Session safely terminated. Run the same command to resume copy again.")
}

func (s sessionV8) copyCloseAndDie(sessionFlag bool) {
	if sessionFlag {
		s.Close()
		console.Fatalln("Command terminated safely. Run this command to resume copy again.")
	} else {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		s.DataFP.Close() // ignore error.
	}
}

// Create a factory function to simplify checking if
// object was last operated on.
func isLastFactory(lastURL string) func(string) bool {
	last := true // closure
	return func(sourceURL string) bool {
		if sourceURL == "" {
			fatalIf(errInvalidArgument().Trace(), "Empty source argument passed.")
		}
		if lastURL == "" {
			return false
		}

		if last {
			if lastURL == sourceURL {
				last = false // from next call onwards we say false.
			}
			return true
		}
		return false
	}
}
