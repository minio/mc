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
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/pkg/console"
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

	if resp.TLS != nil {
		printTLSCertInfo(resp.TLS)
	}

	return err
}
