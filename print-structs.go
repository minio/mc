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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// ErrorMessage container for message reason encapsulation
type ErrorMessage struct {
	Message string
	Error   error
}

func (e ErrorMessage) String() string {
	var message string
	if e.Error != nil {
		switch e.Error.(type) {
		case iodine.Error:
			reason := "Reason: " + iodine.ToError(e.Error).Error()
			message = e.Message + ", " + reason
		default:
			reason := "Reason: " + e.Error.Error()
			message = e.Message + ", " + reason
		}
	}
	return message
}

// Content container for content message structure
type Content struct {
	Filetype string `json:"ContentType"`
	Time     string `json:"LastModified"`
	Size     string `json:"Size"`
	Name     string `json:"Name"`
}

func (c Content) String() string {
	message := console.Time("[%s] ", c.Time)
	message = message + console.Size("%6s ", c.Size)
	message = func() string {
		if c.Filetype == "inode/directory" {
			return message + console.Dir("%s", c.Name)
		}
		return message + console.File("%s", c.Name)
	}()
	return message
}
