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
	"os"
	"path/filepath"
	"regexp"
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
	URLs        []string        `json:"args"`
	Files       map[string]bool `json:"files"`

	Lock *sync.Mutex `json:"-"`
}

func (s sessionV1) String() string {
	message := console.Time("[%s] ", s.Started.Local().Format(printDate))
	message = message + console.SessionID("%s", s.SessionID)
	message = message + console.Command(" [%s %s]", s.CommandType, strings.Join(s.URLs, " "))
	return message
}

// loadSession - reads session file if exists and re-initiates internal variables
func loadSessionV1(sid string) (*sessionV1, error) {
	if !isSessionDirExists() {
		return nil, NewIodine(iodine.New(errInvalidArgument{}, nil))
	}
	sessionFile := getSessionFileV1(sid)

	_, err := os.Stat(sessionFile)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}
	s := new(sessionV1)
	s.Version = "1.0.0"
	// map of command and files copied
	s.URLs = nil
	s.Lock = new(sync.Mutex)
	s.Files = make(map[string]bool)
	qs, err := quick.New(s)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}
	return qs.Data().(*sessionV1), nil
}

func getSessionIDsV1() (sids []string) {
	sessionList, err := filepath.Glob(getSessionDir() + "/*")
	if err != nil {
		console.Fatalf("Unable to list session directory ‘%s’, %s", getSessionDir(), NewIodine(iodine.New(err, nil)))
	}

	for _, path := range sessionList {
		sidReg := regexp.MustCompile("^[a-zA-Z]{8}$")
		sid := filepath.Base(path)
		if sidReg.Match([]byte(sid)) {
			sessionV1, err := loadSessionV1(sid)
			if err != nil {
				console.Fatalf("Unable to load session ‘%s’, %s", sid, NewIodine(iodine.New(err, nil)))
			}
			if sessionV1.Version != "1.0.0" {
				continue
			}
			sids = append(sids, sid)
		}
	}
	return sids
}

func getSessionFileV1(sid string) string {
	return filepath.Join(getSessionDir(), sid)
}
