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
	"encoding/json"
	"fmt"
	"os"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// ErrorMessage container for error messages
type ErrorMessage struct {
	Message   string             `json:"message"`
	Cause     error              `json:"cause"`
	Type      string             `json:"type"`
	CallTrace []probe.TracePoint `json:"trace,omitempty"`
	SysInfo   map[string]string  `json:"sysinfo"`
}

// fatalIf wrapper function which takes error and selectively prints stack frames if available on debug
func fatalIf(err *probe.Error, msg string) {
	if err == nil {
		return
	}
	if globalJSONFlag {
		errorMessage := ErrorMessage{
			Message: msg,
			Type:    "fatal",
			Cause:   err.ToGoError(),
			SysInfo: err.SysInfo,
		}
		if globalDebugFlag {
			errorMessage.CallTrace = err.CallTrace
		}
		json, err := json.Marshal(errorMessage)
		if err != nil {
			console.Fatalln(probe.NewError(err))
		}
		console.Println(string(json))
		os.Exit(1)
	}
	if !globalDebugFlag {
		console.Fatalln(fmt.Sprintf("%s %s", msg, err.ToGoError()))
	}
	console.Fatalln(fmt.Sprintf("%s %s", msg, err))
}

// errorIf synonymous with fatalIf but doesn't exit on error != nil
func errorIf(err *probe.Error, msg string) {
	if err == nil {
		return
	}
	if globalJSONFlag {
		errorMessage := ErrorMessage{
			Message: msg,
			Type:    "error",
			Cause:   err.ToGoError(),
			SysInfo: err.SysInfo,
		}
		if globalDebugFlag {
			errorMessage.CallTrace = err.CallTrace
		}
		json, err := json.Marshal(errorMessage)
		if err != nil {
			console.Fatalln(probe.NewError(err))
		}
		console.Println(string(json))
		return
	}
	if !globalDebugFlag {
		console.Errorln(fmt.Sprintf("%s %s", msg, err.ToGoError()))
		return
	}
	console.Errorln(fmt.Sprintf("%s %s", msg, err))
}
