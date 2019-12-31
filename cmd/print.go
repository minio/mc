/*
 * MinIO Client (C) 2014, 2015 MinIO, Inc.
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

package cmd

import (
	"github.com/minio/minio/pkg/console"
)

// message interface for all structured messages implementing JSON(), String() methods.
type message interface {
	JSON() string
	String() string
}

// printMsg prints message string or JSON structure depending on the type of output console.
func printMsg(msg message) {
	var msgStr string
	if !globalJSON {
		msgStr = msg.String()
	} else {
		msgStr = msg.JSON()
	}
	console.Println(msgStr)
}
