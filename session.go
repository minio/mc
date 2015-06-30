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
	"sync"
	"time"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

type sessionV1 struct {
	Version     string          `json:"version"`
	Started     time.Time       `json:"started"`
	CommandType string          `json:"command-type"`
	SessionID   string          `json:"session-id"`
	RootPath    string          `json:"working-directory"`
	URLs        []string        `json:"args"`
	Files       map[string]bool `json:"files"`

	Lock *sync.Mutex `json:"-"`
}

func (s sessionV1) String() string {
	message := console.SessionID("%s -> ", s.SessionID)
	message = message + console.Time("[%s]", s.Started.Local().Format(printDate))
	message = message + console.Command(" %s %s", s.CommandType, strings.Join(s.URLs, " "))
	return message
}

func isSessionDirExists() bool {
	sdir, err := getSessionDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(sdir)
	if err != nil {
		return false
	}
	return true
}

func createSessionDir() error {
	sdir, err := getSessionDir()
	if err != nil {
		return iodine.New(err, nil)
	}
	if err := os.MkdirAll(sdir, 0700); err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

func getSessionDir() (string, error) {
	mcConfigDir, err := getMcConfigDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return filepath.Join(mcConfigDir, sessionDir), nil
}

var mcCurrentSessionVersion = mcCurrentConfigVersion
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

func newSessionV1() (config quick.Config, err error) {
	s := new(sessionV1)
	s.Version = mcCurrentSessionVersion
	// map of command and files copied
	s.URLs = nil
	s.Started = time.Now().UTC()
	s.Files = make(map[string]bool)
	s.Lock = new(sync.Mutex)
	s.SessionID = newSID(8)
	return quick.New(s)
}

func getSessionFile(sid string) (string, error) {
	sdir, err := getSessionDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return filepath.Join(sdir, sid), nil
}

// save a session
func saveSession(s *sessionV1) error {
	sessionFile, err := getSessionFile(s.SessionID)
	if err != nil {
		return err
	}
	qs, err := quick.New(s)
	if err != nil {
		return err
	}
	return qs.Save(sessionFile)
}

// provides a new session
func newSession() (*sessionV1, error) {
	qs, err := newSessionV1()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return qs.Data().(*sessionV1), nil
}

// loadSession - reads session file if exists and re-initiates internal variables
func loadSession(sid string) (*sessionV1, error) {
	if !isSessionDirExists() {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	sessionFile, err := getSessionFile(sid)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	_, err = os.Stat(sessionFile)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	s := new(sessionV1)
	s.Version = mcCurrentSessionVersion
	// map of command and files copied
	s.URLs = nil
	s.Lock = new(sync.Mutex)
	s.Files = make(map[string]bool)
	qs, err := quick.New(s)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return qs.Data().(*sessionV1), nil

}
