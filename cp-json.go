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

	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
)

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
	copyMessageBytes, err := json.Marshal(c)
	fatalIf(probe.NewError(err), "Failed to marshal copy message.")

	return string(copyMessageBytes) + "\n"
}
