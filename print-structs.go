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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/pkg/console"
)

// ErrorMessage json container for error messages
type ErrorMessage struct {
	*probe.Error
}

func (e ErrorMessage) String() string {
	if !globalJSONFlag {
		return e.Cause.Error()
	}
	if !globalDebugFlag {
		errorMessageBytes, err := json.MarshalIndent(e.Cause, "", "\t")
		if err != nil {
			panic(err)
		}
		return console.JSON(string(errorMessageBytes) + "\n")
	} else {
		errorMessageBytes, err := json.MarshalIndent(e, "", "\t")
		if err != nil {
			panic(err)
		}
		return console.JSON(string(errorMessageBytes) + "\n")
	}
}

// DiffJSONMessage json container for diff messages
type DiffJSONMessage struct {
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

// ContentMessage container for content message structure
type ContentMessage struct {
	Filetype string `json:"type"`
	Time     string `json:"last-modified"`
	Size     string `json:"size"`
	Name     string `json:"name"`
}

// String string printer for Content metadata
func (c ContentMessage) String() string {
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
	jsonMessageBytes, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(jsonMessageBytes) + "\n")
}

// CopyMessage container for file copy messages
type CopyMessage struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Length int64  `json:"length"`
}

// String string printer for copy message
func (c CopyMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("‘%s’ -> ‘%s’\n", c.Source, c.Target)
	}
	copyMessageBytes, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(copyMessageBytes) + "\n")
}

// MirrorMessage container for file mirror messages
type MirrorMessage struct {
	Source  string   `json:"source"`
	Targets []string `json:"targets"`
	Length  int64    `json:"length"`
}

// String string printer for mirror message
func (s MirrorMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("‘%s’ -> ‘%s’\n", s.Source, s.Targets)
	}
	mirrorMessageBytes, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		panic(err)
	}
	return console.JSON(string(mirrorMessageBytes) + "\n")
}

// ShareMessage container for share messages
type ShareMessage struct {
	Expires      time.Duration `json:"expire-seconds"`
	PresignedURL string        `json:"presigned-url"`
}

// String string printer for share message
func (s ShareMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("Succesfully generated shared URL with expiry %s, please share: %s\n", s.Expires, s.PresignedURL)
	}
	shareMessageBytes, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		panic(err)
	}
	// json encoding escapes ampersand into its unicode character which is not usable directly for share and fails with cloud storage.
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	return console.JSON("%s\n", string(shareMessageBytes))
}
