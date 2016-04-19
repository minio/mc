/*
 * Minio Client (C) 2016 Minio, Inc.
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
	"time"

	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

/////////////////// Session V6 ///////////////////
// sessionV6Header for resumable sessions.
type sessionV6Header struct {
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

func loadSessionV6Header(sid string) (*sessionV6Header, *probe.Error) {
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

	sV6Header := &sessionV6Header{}
	sV6Header.Version = "6"
	qs, err := quick.New(sV6Header)
	if err != nil {
		return nil, err.Trace(sid, sV6Header.Version)
	}
	err = qs.Load(sessionFile)
	if err != nil {
		return nil, err.Trace(sid, sV6Header.Version)
	}

	sV6Header = qs.Data().(*sessionV6Header)
	return sV6Header, nil
}

/////////////////// Session V7 ///////////////////
// RESERVED FOR FUTURE
