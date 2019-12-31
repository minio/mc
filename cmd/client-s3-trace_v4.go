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
	"regexp"
	"strings"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/minio/pkg/console"
)

// traceV4 - tracing structure for signature version '4'.
type traceV4 struct{}

// newTraceV4 - initialize Trace structure
func newTraceV4() httptracer.HTTPTracer {
	return traceV4{}
}

// Request - Trace HTTP Request
func (t traceV4) Request(req *http.Request) (err error) {
	origAuth := req.Header.Get("Authorization")

	printTrace := func() error {
		reqTrace, rerr := httputil.DumpRequestOut(req, false) // Only display header
		if rerr == nil {
			console.Debug(string(reqTrace))
		}
		return rerr
	}

	if strings.TrimSpace(origAuth) != "" {
		// Authorization (S3 v4 signature) Format:
		// Authorization: AWS4-HMAC-SHA256 Credential=AKIAJNACEGBGMXBHLEZA/20150524/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=bbfaa693c626021bcb5f911cd898a1a30206c1fad6bad1e0eb89e282173bd24c

		// Strip out accessKeyID from: Credential=<access-key-id>/<date>/<aws-region>/<aws-service>/aws4_request
		regCred := regexp.MustCompile("Credential=([A-Z0-9]+)/")
		newAuth := regCred.ReplaceAllString(origAuth, "Credential=**REDACTED**/")

		// Strip out 256-bit signature from: Signature=<256-bit signature>
		regSign := regexp.MustCompile("Signature=([[0-9a-f]+)")
		newAuth = regSign.ReplaceAllString(newAuth, "Signature=**REDACTED**")

		// Set a temporary redacted auth
		req.Header.Set("Authorization", newAuth)

		err = printTrace()

		// Undo
		req.Header.Set("Authorization", origAuth)
	} else {
		err = printTrace()
	}
	return err
}

// Response - Trace HTTP Response
func (t traceV4) Response(resp *http.Response) (err error) {
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
