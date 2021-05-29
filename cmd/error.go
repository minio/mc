// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
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
	fatal(err, msg, data...)
}

func fatal(err *probe.Error, msg string, data ...interface{}) {
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
		json, e := json.MarshalIndent(struct {
			Status string       `json:"status"`
			Error  errorMessage `json:"error"`
		}{
			Status: "error",
			Error:  errorMsg,
		}, "", " ")
		if e != nil {
			console.Fatalln(probe.NewError(e))
		}
		console.Println(string(json))
		console.Fatalln()
	}

	msg = fmt.Sprintf(msg, data...)
	errmsg := err.String()
	if !globalDebug {
		e := err.ToGoError()
		if errors.Is(e, context.Canceled) {
			// This will replace context canceled error message
			// that the user is seeing to a better one.
			e = errors.New("Canceling upon user request")
		}
		errmsg = e.Error()
	}

	// Remove unnecessary leading spaces in generic/detailed error messages
	msg = strings.TrimSpace(msg)
	errmsg = strings.TrimSpace(errmsg)

	// Add punctuations when needed
	if len(errmsg) > 0 && len(msg) > 0 {
		if msg[len(msg)-1] != ':' && msg[len(msg)-1] != '.' {
			// The detailed error message starts with a capital letter,
			// we should then add '.', otherwise add ':'.
			if unicode.IsUpper(rune(errmsg[0])) {
				msg += "."
			} else {
				msg += ":"
			}
		}
		// Add '.' to the detail error if not found
		if errmsg[len(errmsg)-1] != '.' {
			errmsg += "."
		}
	}

	console.Fatalln(fmt.Sprintf("%s %s", msg, errmsg))
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
		json, e := json.MarshalIndent(struct {
			Status string       `json:"status"`
			Error  errorMessage `json:"error"`
		}{
			Status: "error",
			Error:  errorMsg,
		}, "", " ")
		if e != nil {
			console.Fatalln(probe.NewError(e))
		}
		console.Println(string(json))
		return
	}
	msg = fmt.Sprintf(msg, data...)
	if !globalDebug {
		e := err.ToGoError()
		if errors.Is(e, context.Canceled) {
			// This will replace context canceled error message
			// that the user is seeing to a better one.
			e = errors.New("Canceling upon user request")
		}
		console.Errorln(fmt.Sprintf("%s %s", msg, e))
		return
	}
	console.Errorln(fmt.Sprintf("%s %s", msg, err))
}
