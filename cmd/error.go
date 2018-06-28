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

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// causeMessage container for golang error messages
type causeMessage struct {
	Message string `json:"message"`
	Error   error  `json:"error"`
}

// errorMessage container for error messages
type errorMessage struct {
	Message   string             `json:"message"`
	Cause     causeMessage       `json:"cause"`
	Type      string             `json:"type"`
	CallTrace []probe.TracePoint `json:"trace,omitempty"`
	SysInfo   map[string]string  `json:"sysinfo"`
}

// fatalIf wrapper function which takes error and selectively prints stack frames if available on debug
func fatalIf(err *probe.Error, msg string, data ...interface{}) {
	if err == nil {
		return
	}
	if globalJSON {
		errorMsg := errorMessage{
			Message: msg,
			Type:    "fatal",
			Cause: causeMessage{
				Message: err.ToGoError().Error(),
				Error:   err.ToGoError(),
			},
			SysInfo: err.SysInfo,
		}
		if globalDebug {
			errorMsg.CallTrace = err.CallTrace
		}
		json, e := json.Marshal(struct {
			Status string       `json:"status"`
			Error  errorMessage `json:"error"`
		}{
			Status: "error",
			Error:  errorMsg,
		})
		if e != nil {
			console.Fatalln(probe.NewError(e))
		}
		console.Println(string(json))
		console.Fatalln()
	}
	msg = fmt.Sprintf(msg, data...)
	if !globalDebug {
		console.Fatalln(fmt.Sprintf("%s %s", msg, err.ToGoError()))
	}
	console.Fatalln(fmt.Sprintf("%s %s", msg, err))
}

// Exit coder wraps cli new exit error with a
// custom exitStatus number. cli package requires
// an error with `cli.ExitCoder` compatibility
// after an action. Which woud allow cli package to
// exit with the specified `exitStatus`.
func exitStatus(status int) error {
	return cli.NewExitError("", status)
}

// errorIf synonymous with fatalIf but doesn't exit on error != nil
func errorIf(err *probe.Error, msg string, data ...interface{}) {
	if err == nil {
		return
	}
	if globalJSON {
		errorMsg := errorMessage{
			Message: fmt.Sprintf(msg, data...),
			Type:    "error",
			Cause: causeMessage{
				Message: err.ToGoError().Error(),
				Error:   err.ToGoError(),
			},
			SysInfo: err.SysInfo,
		}
		if globalDebug {
			errorMsg.CallTrace = err.CallTrace
		}
		json, e := json.Marshal(struct {
			Status string       `json:"status"`
			Error  errorMessage `json:"error"`
		}{
			Status: "error",
			Error:  errorMsg,
		})
		if e != nil {
			console.Fatalln(probe.NewError(e))
		}
		console.Println(string(json))
		return
	}
	msg = fmt.Sprintf(msg, data...)
	if !globalDebug {
		console.Errorln(fmt.Sprintf("%s %s", msg, err.ToGoError()))
		return
	}
	console.Errorln(fmt.Sprintf("%s %s", msg, err))
}
