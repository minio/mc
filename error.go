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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Fatal wrapper function which takes error and selectively prints stack frames if available on debug
func Fatal(err error) {
	if err == nil {
		return
	}
	switch e := err.(type) {
	case *probe.Error:
		if e == nil {
			return
		}
		if !globalDebugFlag {
			console.Fatalln(e.ToError())
		}
		console.Fatalln(err)
	default:
		console.Fatalln(err)
	}
}

// Error synonymous with Fatal but doesn't exit on error != nil
func Error(err error) {
	if err == nil {
		return
	}
	switch e := err.(type) {
	case *probe.Error:
		if e == nil {
			return
		}
		if !globalDebugFlag {
			console.Errorln(e.ToError())
			return
		}
		console.Errorln(err)
	default:
		console.Errorln(err)
	}
}
