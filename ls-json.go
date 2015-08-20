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

package main

import (
	"encoding/json"
	"fmt"

	"github.com/minio/mc/pkg/console"
)

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
		message := console.Colorize("Time", fmt.Sprintf("[%s] ", c.Time))
		message = message + console.Colorize("Size", fmt.Sprintf("%6s ", c.Size))
		message = func() string {
			if c.Filetype == "folder" {
				return message + console.Colorize("Dir", fmt.Sprintf("%s", c.Name))
			}
			return message + console.Colorize("File", fmt.Sprintf("%s", c.Name))
		}()
		return message + "\n"
	}
	jsonMessageBytes, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(jsonMessageBytes) + "\n"
}
