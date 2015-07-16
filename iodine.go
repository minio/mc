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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio/pkg/iodine"
)

// Error is the iodine error which contains a pointer to the original error
// and stack traces.
type Error struct {
	EmbeddedError error `json:"-"`
	ErrorMessage  string
	ErrorType     string

	Stack []StackEntry
}

// StackEntry contains the entry in the stack trace
type StackEntry struct {
	Host string
	File string
	Func string
	Line int
	Data map[string]string
}

// NewIodine - instantiate an error, turning it into an iodine error.
// Adds an initial stack trace.
func NewIodine(err error, data map[string]string) error {
	_, iodineFile, _, _ := runtime.Caller(0)
	iodineFile = filepath.Dir(iodineFile)         // trim iodine.go
	iodineFile = filepath.Dir(iodineFile)         // trim iodine
	iodineFile = filepath.Dir(iodineFile)         // trim pkg
	iodineFile = filepath.Dir(iodineFile)         // trim minio
	iodineFile = filepath.Dir(iodineFile)         // trim minio
	gopathSource = filepath.Dir(iodineFile) + "/" // trim github.com

	if err != nil {
		entry := createStackEntry()
		var newErr Error

		// check if error is wrapped
		switch typedError := err.(type) {
		case Error:
			{
				newErr = typedError
			}
		default:
			{
				newErr = Error{
					EmbeddedError: err,
					ErrorMessage:  err.Error(),
					ErrorType:     reflect.TypeOf(err).String(),
					Stack:         []StackEntry{},
				}
			}
		}
		for k, v := range data {
			entry.Data[k] = v
		}
		newErr.Stack = append(newErr.Stack, entry)
		return newErr
	}
	return nil
}

var gopathSource string

// createStackEntry - create stack entries
func createStackEntry() StackEntry {
	host, _ := os.Hostname()
	pc, file, line, _ := runtime.Caller(2)
	function := runtime.FuncForPC(pc).Name()
	_, function = filepath.Split(function)
	file = strings.TrimPrefix(file, gopathSource) // trim gopathSource from file

	data := make(map[string]string)
	for k, v := range getIodineSystemData() {
		data[k] = v
	}

	entry := StackEntry{
		Host: host,
		File: file,
		Func: function,
		Line: line,
		Data: data,
	}
	return entry
}

func getIodineSystemData() map[string]string {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	return map[string]string{
		"sys.host":               host,
		"sys.os":                 runtime.GOOS,
		"sys.arch":               runtime.GOARCH,
		"sys.go":                 runtime.Version(),
		"sys.cpus":               strconv.Itoa(runtime.NumCPU()),
		"sys.mem.used":           humanize.Bytes(memstats.Alloc),
		"sys.mem.allocated":      humanize.Bytes(memstats.TotalAlloc),
		"sys.mem.heap.used":      humanize.Bytes(memstats.HeapAlloc),
		"sys.mem.heap.allocated": humanize.Bytes(memstats.HeapSys),
	}
}

// ToError returns the input if it is not an iodine error. It returns the embedded error if it is an iodine error. If nil, returns nil.
func ToError(err error) error {
	switch err := err.(type) {
	case nil:
		{
			return nil
		}
	case Error:
		{
			if err.EmbeddedError != nil {
				return err.EmbeddedError
			}
			return errors.New(err.ErrorMessage)
		}
	default:
		{
			return err
		}
	}
}

// EmitHumanReadable returns a human readable error message
func (err Error) EmitHumanReadable() string {
	var errorBuffer bytes.Buffer
	if globalDebugFlag {
		fmt.Fprint(&errorBuffer, err.ErrorMessage)
		for i, entry := range err.Stack {
			prettyData, _ := json.Marshal(entry.Data)
			fmt.Fprintln(&errorBuffer, "-", i, entry.Host+":"+entry.File+":"+strconv.Itoa(entry.Line)+" "+entry.Func+"():", string(prettyData))
		}
	} else {
		fmt.Fprint(&errorBuffer, iodine.ToError(err.EmbeddedError))
	}
	return string(errorBuffer.Bytes())
}

// Emits the original error message
func (err Error) Error() string {
	return err.EmitHumanReadable()
}
