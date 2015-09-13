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

import "github.com/minio/mc/pkg/console"

// Message interface for all structured messages implementing JSON(), String() methods
type Message interface {
	JSON() string
	String() string
}

// Prints print wrapper for Message interface, implementing both JSON() and String() methods
func Prints(format string, message Message) {
	if !globalJSONFlag {
		console.Printf(format, message.String())
		return
	}
	console.Printf(format, message.JSON())
}
