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
	"sync"
	"time"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

// migrateSessionV1ToV2 migrates all session files from v1 to v2.
// This function should be called from the main early on.
func migrateSessionV1ToV2() {
	for _, sid := range getSessionIDsV1() {
		err := os.Remove(getSessionFileV1(sid))
		if err != nil {
			console.Fatalf("Migration failed. Unable to remove old session file %s. %s\n", getSessionFileV1(sid), NewIodine(iodine.New(err, nil)))
		}
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

// sessionV2
type sessionV2 struct {
	Header    *sessionV2Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *os.File
	sigCh     bool
}

// newSessionV2 provides a new session
func newSessionV2() *sessionV2 {
	if !isMcConfigExists() {
		console.Fatalf("Please run \"mc config generate\". %s\n", errNotConfigured{})
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
		console.Fatalf("Unable to create session data file \""+getSessionDataFile(s.SessionID)+"\". %s\n", err)
	}
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
func (s *sessionV2) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.DataFP.Sync()
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}

	qs, err := quick.New(s.Header)
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}

	return qs.Save(getSessionFile(s.SessionID))
}

// Close ends this session and removes all associated session files.
func (s *sessionV2) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.DataFP.Close()
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}

	err = os.Remove(getSessionDataFile(s.SessionID))
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}

	err = os.Remove(getSessionFile(s.SessionID))
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}

	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}
	return nil
}

// loadSession - reads session file if exists and re-initiates internal variables
func loadSessionV2(sid string) (*sessionV2, error) {
	if !isSessionDirExists() {
		return nil, NewIodine(iodine.New(errInvalidArgument{}, nil))
	}
	sessionFile := getSessionFile(sid)

	_, err := os.Stat(sessionFile)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}

	s := &sessionV2{}
	s.Header = &sessionV2Header{}
	s.SessionID = sid
	s.Header.Version = "1.1.0"
	qs, err := quick.New(s.Header)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}

	s.mutex = new(sync.Mutex)
	s.Header = qs.Data().(*sessionV2Header)

	s.DataFP, err = os.Open(getSessionDataFile(s.SessionID))
	if err != nil {
		console.Fatalf("Unable to open session data file \""+getSessionDataFile(s.SessionID)+"\". %s", NewIodine(iodine.New(errNotConfigured{}, nil)))
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
			console.Fatalf("Empty source URL passed to isCopied() function. %s\n", NewIodine(iodine.New(errUnexpected{}, nil)))
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
