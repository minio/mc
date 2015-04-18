/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"errors"
	"net/http"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
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
	if t.Trace != nil {
		err = t.Trace.Request(req)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
	}

	if t.Transport == nil {
		return nil, iodine.New(client.InvalidArgument{Err: errors.New("invalid argument")}, nil)
	}

	res, err = t.Transport.RoundTrip(req)
	if err != nil {
		return res, iodine.New(err, nil)
	}

	if t.Trace != nil {
		t.Trace.Response(res)
	}

	return res, iodine.New(err, nil)
}

// GetNewTraceTransport returns a traceable transport
func GetNewTraceTransport(trace HTTPTracer, transport http.RoundTripper) RoundTripTrace {
	return RoundTripTrace{Trace: trace,
		Transport: transport}
}
