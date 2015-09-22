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
	"time"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
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
		return nil, client.InvalidArgument{}
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
func GetNewTraceTransport(trace HTTPTracer, transport http.RoundTripper) RoundTripTrace {
	return RoundTripTrace{Trace: trace,
		Transport: transport}
}
