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

import "encoding/json"

// DiffMessage json container for diff messages
type DiffMessage struct {
	FirstURL  string `json:"first"`
	SecondURL string `json:"second"`
	Diff      string `json:"diff"`
}

func (d DiffMessage) String() string {
	if !globalJSONFlag {
		var message string
		if d.Diff == "Only-in" {
			message = "‘" + d.FirstURL + "’ Only in ‘" + d.SecondURL + "’\n"
		}
		if d.Diff == "Type" {
			message = d.FirstURL + " and " + d.SecondURL + " differs in type.\n"
		}
		if d.Diff == "Size" {
			message = d.FirstURL + " and " + d.SecondURL + " differs in size.\n"
		}
		return message
	}
	diffJSONBytes, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}
	return string(diffJSONBytes) + "\n"
}
