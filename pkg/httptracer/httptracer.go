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

// Package httptracer implements http tracing functionality
package httptracer

import (
	"errors"
	"net/http"
	"time"

	"github.com/minio/pkg/console"
)

// HTTPTracer provides callback hook mechanism for HTTP transport.
type HTTPTracer interface {
	Request(req *http.Request) error
	Response(res *http.Response) error
}

// RoundTripTrace interposes HTTP transport requests and respsonses using HTTPTracer hooks
type RoundTripTrace struct {
	Trace     HTTPTracer        // User provides callback methods
	Transport http.RoundTripper // HTTP transport that needs to be intercepted
}

// RoundTrip executes user provided request and response hooks for each HTTP call.
func (t RoundTripTrace) RoundTrip(req *http.Request) (res *http.Response, err error) {
	timeStamp := time.Now()

	if t.Transport == nil {
		return nil, errors.New("Invalid Argument")
	}

	res, err = t.Transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	if t.Trace != nil {
		err = t.Trace.Request(req)
		if err != nil {
			return nil, err
		}

		err = t.Trace.Response(res)
		if err != nil {
			return nil, err
		}
		console.Debugln("Response Time: ", time.Since(timeStamp).String()+"\n")
	}
	return res, err
}

// GetNewTraceTransport returns a traceable transport
//
// Takes first argument a custom tracer which implements Response, Request() conforming to HTTPTracer interface.
// Another argument can be a default transport or a custom http.RoundTripper implementation
func GetNewTraceTransport(trace HTTPTracer, transport http.RoundTripper) RoundTripTrace {
	return RoundTripTrace{Trace: trace,
		Transport: transport}
}
