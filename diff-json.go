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
	diffJSONBytes, err := json.Marshal(diffMessage)
	if err != nil {
		panic(err)
	}
	return string(diffJSONBytes) + "\n"
}
