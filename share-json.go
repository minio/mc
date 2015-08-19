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
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

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
	shareMessageBytes, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	// json encoding escapes ampersand into its unicode character which is not usable directly for share
	// and fails with cloud storage. convert them back so that they are usable
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	return fmt.Sprintf("%s\n", string(shareMessageBytes))
}
