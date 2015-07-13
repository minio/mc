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

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// ErrorMessage container for error reason encapsulation
type ErrorMessage struct {
	Message string `json:"-"`
	Error   error  `json:"error"`
}

// String string printer for error message
func (e ErrorMessage) String() string {
	if !globalJSONFlag {
		var message string
		if e.Error != nil {
			switch e.Error.(type) {
			case iodine.Error:
				reason := iodine.ToError(e.Error).Error()
				message = reason
			default:
				reason := e.Error.Error()
				message = reason
			}
		}
		return message
	}
	eBytes, err := json.Marshal(iodine.ToError(e.Error))
	if err != nil {
		panic(err)
	}
	return string(eBytes)
}

// Content container for content message structure
type Content struct {
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
			if c.Filetype == "directory" {
				return message + console.Dir("%s", c.Name)
			}
			return message + console.File("%s", c.Name)
		}()
		return message
	}
	cBytes, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(cBytes)
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
		return fmt.Sprintf("‘%s’ -> ‘%s’", c.Source, c.Target)
	}
	cBytes, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(cBytes)
}

// SyncMessage container for file sync messages, inherits CopyMessage
type SyncMessage struct {
	Source  string   `json:"source"`
	Targets []string `json:"targets"`
	Length  int64    `json:"length"`
}

// String string printer for sync message
func (s SyncMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("‘%s’ -> ‘%s’", s.Source, s.Targets)
	}
	sBytes, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(sBytes)
}
