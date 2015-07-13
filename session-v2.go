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
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

// migrateSessionV1ToV2 migrates all session files from v1 to v2. This
// function should be called from the main early on.
func migrateSessionV1ToV2() {
	// TODO: Fill it in.
	for _, sid := range getSessionIDsV1() {
		err := os.Remove(getSessionFileV1(sid))
		if err != nil {
			console.Fatalf("Migration failed. Unable to remove old session file %s. %s\n", getSessionFileV1(sid), iodine.New(err, nil))
		}
	}
}

type sessionV2Header struct {
	Version      string    `json:"version"`
	When         time.Time `json:"time"`
	RootPath     string    `json:"working-directory"`
	CommandType  string    `json:"command-type"`
	CommandArgs  []string  `json:"cmd-args"`
	LastCopied   string    `json:"last-copied"`
	TotalBytes   int64     `json:"total-bytes"`
	TotalObjects int       `json:"total-objects"`
}

type sessionV2 struct {
	Header    *sessionV2Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *os.File
	sigCh     bool
}

// provides a new session
func newSessionV2() *sessionV2 {
	// TODO: Blindly create .mc and session dirs at init and remove these checks -ab.
	if !isMcConfigExists() {
		console.Fatals(ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errNotConfigured{}, nil),
		})
	}

	if !isSessionDirExists() {
		if err := createSessionDir(); err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
	}

	s := &sessionV2{}
	s.Header = &sessionV2Header{}
	s.Header.Version = "1.1.0"
	// map of command and files copied
	s.Header.CommandArgs = nil
	s.Header.When = time.Now().UTC()
	s.mutex = new(sync.Mutex)
	s.SessionID = newSID(8)
	var err error
	s.DataFP, err = os.Create(getSessionDataFile(s.SessionID))
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: "Unable to create session data file \"" + getSessionDataFile(s.SessionID) + "\"",
			Error:   iodine.New(errNotConfigured{}, nil),
		})
	}

	return s
}

func (s *sessionV2) String() string {
	message := console.SessionID("%s -> ", s.SessionID)
	message = message + console.Time("[%s]", s.Header.When.Local().Format(printDate))
	message = message + console.Command(" %s %s", s.Header.CommandType, strings.Join(s.Header.CommandArgs, " "))
	return message
}

func (s *sessionV2) Info() {
	console.Infoln("Session terminated. To resume session type ‘mc session resume " + s.SessionID + "’")
}

// NewDataReader provides reader interface to session data file.
func (s *sessionV2) HasData() bool {
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

// save this session
func (s *sessionV2) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.DataFP.Sync()
	if err != nil {
		return iodine.New(err, nil)
	}

	qs, err := quick.New(s.Header)
	if err != nil {
		return iodine.New(err, nil)
	}

	return qs.Save(getSessionFile(s.SessionID))
}

// Close ends this session and removes all associated session files.
func (s *sessionV2) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := os.Remove(getSessionDataFile(s.SessionID))
	if err != nil {
		return iodine.New(err, nil)
	}

	err = os.Remove(getSessionFile(s.SessionID))
	if err != nil {
		return iodine.New(err, nil)
	}

	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// loadSession - reads session file if exists and re-initiates internal variables
func loadSessionV2(sid string) (*sessionV2, error) {
	if !isSessionDirExists() {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	sessionFile := getSessionFile(sid)

	_, err := os.Stat(sessionFile)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	s := &sessionV2{}
	s.Header = &sessionV2Header{}
	s.SessionID = sid
	s.Header.Version = "1.1.0"
	qs, err := quick.New(s.Header)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	s.mutex = new(sync.Mutex)
	s.Header = qs.Data().(*sessionV2Header)

	s.DataFP, err = os.Open(getSessionDataFile(s.SessionID))
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: "Unable to open session data file \"" + getSessionDataFile(s.SessionID) + "\"",
			Error:   iodine.New(errNotConfigured{}, nil),
		})
	}

	return s, nil
}

// Create a factory function to simplify checking if an
// object has been copied or not.
// isCopied(URL) -> true or false
func isCopiedFactory(lastCopied string) func(string) bool {
	copied := true // closure
	return func(sourceURL string) bool {
		if sourceURL == "" {
			console.Fatalln("Empty source URL passed to isCopied() function. Please report this bug.")
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
