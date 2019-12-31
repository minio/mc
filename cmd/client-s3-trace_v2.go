/*
 * MinIO Client (C) 2015 MinIO, Inc.
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
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/minio/pkg/console"
)

// traceV2 - tracing structure for signature version '2'.
type traceV2 struct{}

// newTraceV2 - initialize Trace structure
func newTraceV2() httptracer.HTTPTracer {
	return traceV2{}
}

// Request - Trace HTTP Request
func (t traceV2) Request(req *http.Request) (err error) {
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
func (t traceV2) Response(resp *http.Response) (err error) {
	var respTrace []byte
	// For errors we make sure to dump response body as well.
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusPartialContent &&
		resp.StatusCode != http.StatusNoContent {
		respTrace, err = httputil.DumpResponse(resp, true)
	} else {
		respTrace, err = httputil.DumpResponse(resp, false)
	}
	if err == nil {
		console.Debug(string(respTrace))
	}

	if globalInsecure && resp.TLS != nil {
		dumpTLSCertificates(resp.TLS)
	}

	return err
}
