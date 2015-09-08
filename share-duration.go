/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// shareDuration extended version of time.Duration implementing .Days() for convenience
type shareDuration struct {
	duration     time.Duration
	presignedURL string
}

func (s shareDuration) Days() float64 {
	return s.duration.Hours() / 24
}

func (s shareDuration) Seconds() float64 {
	return s.duration.Seconds()
}

func (s shareDuration) Hours() float64 {
	return s.duration.Hours()
}

func (s shareDuration) GetDuration() time.Duration {
	return s.duration
}

func (s shareDuration) String() string {
	if !globalJSONFlag {
		durationString := func() string {
			if s.duration.Hours() > 24 {
				return fmt.Sprintf("%dd", int64(s.Days()))
			}
			return s.duration.String()
		}
		return console.Colorize("Share", fmt.Sprintf("Expiry: %s\n   URL: %s", durationString(), s.presignedURL))
	}
	shareMessageBytes, err := json.Marshal(struct {
		Expires      time.Duration `json:"expireSeconds"`
		PresignedURL string        `json:"presignedURL"`
	}{
		Expires:      time.Duration(s.Seconds()),
		PresignedURL: s.presignedURL,
	})
	fatalIf(probe.NewError(err), "Failed to marshal into JSON.")

	// json encoding escapes ampersand into its unicode character which is not usable directly for share
	// and fails with cloud storage. convert them back so that they are usable
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	return string(shareMessageBytes)
}
