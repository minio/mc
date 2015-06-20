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
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

type sessionV1 struct {
	Version   string
	SessionID string
	Command   string
	Files     []string
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

// newUUID generates a random UUID according to RFC 4122
func newUUID() string {
	uuid := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		panic(err)
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func newSessionV1() (config quick.Config, err error) {
	s := new(sessionV1)
	s.Version = mcCurrentSessionVersion
	// map of command and files copied
	s.Command = ""
	s.Files = nil
	s.SessionID = newUUID()
	return quick.New(s)
}

func writeSession(config quick.Config) error {
	if err := createSessionDir(); err != nil {
		return iodine.New(err, nil)
	}
	sdir, err := getSessionDir()
	if err != nil {
		return iodine.New(err, nil)
	}
	if err := config.Save(filepath.Join(sdir, config.Data().(*sessionV1).SessionID)); err != nil {
		return iodine.New(err, nil)
	}
	return nil
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
	if err := writeSession(qs); err != nil {
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
	s.Command = ""
	s.Files = nil
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
