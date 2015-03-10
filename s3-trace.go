/*
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
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
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
)

type s3Trace struct {
	BodyTraceFlag        bool      // Include Body
	RequestTransportFlag bool      // Include additional http.Transport adds such as User-Agent
	Writer               io.Writer // Console device to write to
}

// Trace HTTP Request
func (t s3Trace) Request(req *http.Request) {
	if t.RequestTransportFlag {
		reqTrace, err := httputil.DumpRequestOut(req, t.BodyTraceFlag)
		if err == nil {
			t.trace(reqTrace)
		}
	} else {
		reqTrace, err := httputil.DumpRequest(req, t.BodyTraceFlag)
		if err == nil {
			t.trace(reqTrace)
		}
	}
}

// Trace HTTP Response
func (t s3Trace) Response(res *http.Response) {
	resTrace, err := httputil.DumpResponse(res, t.BodyTraceFlag)
	if err == nil {
		t.trace(resTrace)
	}
}

// Trace HTTP Response
func (t s3Trace) trace(data []byte) {
	if t.Writer != nil {
		fmt.Fprintf(t.Writer, "%s\n", data)
	} else {
		fmt.Printf("%s\n", data)
	}
}
