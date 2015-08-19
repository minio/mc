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
	"strings"

	"github.com/minio/mc/pkg/console"
)

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
	sessionJSONBytes, err := json.Marshal(sessionMesage)
	if err != nil {
		panic(err)
	}
	return string(sessionJSONBytes) + "\n"
}
