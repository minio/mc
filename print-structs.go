/*
 * Minio Client (C) 2015 Minio, Inc.
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

// This package contains all the structs, their method wrappers for printer
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/minio/mc/pkg/console"
)

// DiffJSONMessage json container for diff messages
type DiffJSONMessage struct {
	Version   string `json:"version"`
	FirstURL  string `json:"first"`
	SecondURL string `json:"second"`
	Diff      string `json:"diff"`
}

func (s diffV1) String() string {
	if !globalJSONFlag {
		var message string
		if s.diffType == "Only-in" {
			message = "‘" + s.firstURL + "’ Only in ‘" + s.secondURL + "’\n"
		}
		if s.diffType == "Type" {
			message = s.firstURL + " and " + s.secondURL + " differs in type.\n"
		}
		if s.diffType == "Size" {
			message = s.firstURL + " and " + s.secondURL + " differs in size.\n"
		}
		return message
	}
	diffMessage := DiffJSONMessage{}
	diffMessage.Version = "1.0.0"
	diffMessage.FirstURL = s.firstURL
	diffMessage.SecondURL = s.secondURL
	diffMessage.Diff = s.diffType
	diffJSONBytes, err := json.MarshalIndent(diffMessage, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(diffJSONBytes) + "\n")
}

// SessionJSONMessage json container for session messages
type SessionJSONMessage struct {
	Version     string   `json:"version"`
	SessionID   string   `json:"sessionid"`
	Time        string   `json:"time"`
	CommandType string   `json:"command-type"`
	CommandArgs []string `json:"command-args"`
}

func (s sessionV2) String() string {
	if !globalJSONFlag {
		message := console.SessionID("%s -> ", s.SessionID)
		message = message + console.Time("[%s]", s.Header.When.Local().Format(printDate))
		message = message + console.Command(" %s %s", s.Header.CommandType, strings.Join(s.Header.CommandArgs, " "))
		return message + "\n"
	}
	sessionMesage := SessionJSONMessage{
		Version:     s.Header.Version,
		SessionID:   s.SessionID,
		Time:        s.Header.When.Local().Format(printDate),
		CommandType: s.Header.CommandType,
		CommandArgs: s.Header.CommandArgs,
	}
	sessionJSONBytes, err := json.MarshalIndent(sessionMesage, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(sessionJSONBytes) + "\n")
}

// Content container for content message structure
type Content struct {
	Version  string `json:"version"`
	Filetype string `json:"type"`
	Time     string `json:"last-modified"`
	Size     string `json:"size"`
	Name     string `json:"name"`
}

// String string printer for Content metadata
func (c Content) String() string {
	if !globalJSONFlag {
		message := console.Time("[%s] ", c.Time)
		message = message + console.Size("%6s ", c.Size)
		message = func() string {
			if c.Filetype == "folder" {
				return message + console.Dir("%s", c.Name)
			}
			return message + console.File("%s", c.Name)
		}()
		return message + "\n"
	}
	c.Version = "1.0.0"
	jsonMessageBytes, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(jsonMessageBytes) + "\n")
}

// CopyMessage container for file copy messages
type CopyMessage struct {
	Version string `json:"version"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Length  int64  `json:"length"`
}

// String string printer for copy message
func (c CopyMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("‘%s’ -> ‘%s’\n", c.Source, c.Target)
	}
	c.Version = "1.0.0"
	copyMessageBytes, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(copyMessageBytes) + "\n")
}

// MirrorMessage container for file mirror messages
type MirrorMessage struct {
	Version string   `json:"version"`
	Source  string   `json:"source"`
	Targets []string `json:"targets"`
	Length  int64    `json:"length"`
}

// String string printer for mirror message
func (s MirrorMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("‘%s’ -> ‘%s’\n", s.Source, s.Targets)
	}
	s.Version = "1.0.0"
	mirrorMessageBytes, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(mirrorMessageBytes) + "\n")
}
