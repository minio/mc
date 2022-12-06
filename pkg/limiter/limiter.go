// Copyright (c) 2015-2022 MinIO, Inc.
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

// Package limiter implements throughput upload and download limits via http.RoundTripper
package limiter

import (
	"errors"
	"io"
	"net/http"

	"github.com/juju/ratelimit"
)

type limiter struct {
	upload    *ratelimit.Bucket
	download  *ratelimit.Bucket
	transport http.RoundTripper // HTTP transport that needs to be intercepted
}

func (l limiter) limitReader(r io.Reader, b *ratelimit.Bucket) io.Reader {
	if b == nil {
		return r
	}
	return ratelimit.Reader(r, b)
}

// RoundTrip executes user provided request and response hooks for each HTTP call.
func (l limiter) RoundTrip(req *http.Request) (res *http.Response, err error) {
	if l.transport == nil {
		return nil, errors.New("Invalid Argument")
	}

	type readCloser struct {
		io.Reader
		io.Closer
	}

	if req.Body != nil {
		req.Body = &readCloser{
			Reader: l.limitReader(req.Body, l.upload),
			Closer: req.Body,
		}
	}

	res, err = l.transport.RoundTrip(req)
	if res != nil && res.Body != nil {
		res.Body = &readCloser{
			Reader: l.limitReader(res.Body, l.download),
			Closer: res.Body,
		}
	}

	return res, err
}

// New return a ratelimited transport
func New(uploadLimit, downloadLimit int64, transport http.RoundTripper) http.RoundTripper {
	if uploadLimit == 0 && downloadLimit == 0 {
		return transport
	}

	var (
		uploadBucket   *ratelimit.Bucket
		downloadBucket *ratelimit.Bucket
	)

	if uploadLimit > 0 {
		uploadBucket = ratelimit.NewBucketWithRate(float64(uploadLimit), uploadLimit)
	}

	if downloadLimit > 0 {
		downloadBucket = ratelimit.NewBucketWithRate(float64(downloadLimit), downloadLimit)
	}

	return &limiter{
		upload:    uploadBucket,
		download:  downloadBucket,
		transport: transport,
	}
}
