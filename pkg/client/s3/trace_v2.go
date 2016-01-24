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

package s3

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/httptracer"
)

// TraceV2 - tracing structure
type TraceV2 struct{}

// NewTraceV2 - initialize Trace structure
func NewTraceV2() httptracer.HTTPTracer {
	return TraceV2{}
}

// Request - Trace HTTP Request
func (t TraceV2) Request(req *http.Request) (err error) {
	origAuth := req.Header.Get("Authorization")

	if strings.TrimSpace(origAuth) != "" {
		// Authorization (S3 v2 signature) Format:
		// Authorization: AWS AKIAJVA5BMMU2RHO6IO1:Y10YHUZ0DTUterAUI6w3XKX7Iqk=

		// Set a temporary redacted auth
		req.Header.Set("Authorization", "AWS **REDACTED**:**REDACTED**")

		var reqTrace []byte
		reqTrace, err = httputil.DumpRequestOut(req, false) // Only display header
		if err == nil {
			console.Debug(string(reqTrace))
		}

		// Undo
		req.Header.Set("Authorization", origAuth)
	}
	return err
}

// Response - Trace HTTP Response
func (t TraceV2) Response(res *http.Response) (err error) {
	var resTrace []byte
	// For errors we make sure to dump response body as well.
	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusPartialContent &&
		res.StatusCode != http.StatusNoContent {
		resTrace, err = httputil.DumpResponse(res, true)
	} else {
		// Only display header
		resTrace, err = httputil.DumpResponse(res, false)
	}
	if err == nil {
		console.Debug(string(resTrace))
	}
	return err
}
