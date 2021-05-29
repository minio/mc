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

package cmd

import (
	"os"
	"sync"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/quick"
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
	qs, e := quick.NewConfig(sV6Header, nil)
	if e != nil {
		return nil, probe.NewError(e).Trace(sid, sV6Header.Version)
	}
	e = qs.Load(sessionFile)
	if e != nil {
		return nil, probe.NewError(e).Trace(sid, sV6Header.Version)
	}

	sV6Header = qs.Data().(*sessionV6Header)
	return sV6Header, nil
}

/////////////////// Session V7 ///////////////////
// RESERVED FOR FUTURE

// sessionV7Header for resumable sessions.
type sessionV7Header struct {
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
	TotalObjects       int               `json:"totalObjects"`
}

// sessionV7 resumable session container.
type sessionV7 struct {
	Header    *sessionV7Header
	SessionID string
	mutex     *sync.Mutex
	DataFP    *sessionDataFP
}

// loadSessionV7 - reads session file if exists and re-initiates internal variables
func loadSessionV7(sid string) (*sessionV7, *probe.Error) {
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

	s := &sessionV7{}
	s.Header = &sessionV7Header{}
	s.SessionID = sid
	s.Header.Version = "7"
	qs, e := quick.NewConfig(s.Header, nil)
	if e != nil {
		return nil, probe.NewError(e).Trace(sid, s.Header.Version)
	}
	e = qs.Load(sessionFile)
	if e != nil {
		return nil, probe.NewError(e).Trace(sid, s.Header.Version)
	}

	s.mutex = new(sync.Mutex)
	s.Header = qs.Data().(*sessionV7Header)

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
