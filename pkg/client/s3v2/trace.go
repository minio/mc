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

package s3v2

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/minio/mc/pkg/console"
)

// Trace - tracing structure
type Trace struct {
}

// NewTrace - initialize Trace structure
func NewTrace() HTTPTracer {
	return Trace{}
}

// Request - Trace HTTP Request
func (t Trace) Request(req *http.Request) (err error) {
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
func (t Trace) Response(res *http.Response) (err error) {
	resTrace, err := httputil.DumpResponse(res, false) // Only display header
	if err == nil {
		console.Debug(string(resTrace))
	}
	return err
}
