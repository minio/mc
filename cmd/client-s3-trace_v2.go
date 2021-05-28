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
	"strings"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/pkg/console"
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

	if resp.TLS != nil {
		printTLSCertInfo(resp.TLS)
	}

	return err
}
