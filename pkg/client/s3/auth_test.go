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
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/minio-io/check"
)

type reqAndExpected struct {
	req, expected, host string
}

func req(s string) *http.Request {
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(s)))
	if err != nil {
		panic(fmt.Sprintf("bad request in test: %v (error: %v)", req, err))
	}
	return req
}

func (s *MySuite) TestsignRequest(c *C) {
	r := req("GET /foo HTTP/1.1\n\n")
	config := &Config{
		HostURL:         "localhost:9000",
		AccessKeyID:     "key",
		SecretAccessKey: "secretkey",
		UserAgent:       "Minio/auth_test (mc)",
	}
	url, _ := url.Parse("localhost:9000")
	cl := &s3Client{
		&Meta{
			Config:    config,
			Transport: http.DefaultTransport,
		}, url,
	}
	r.Header.Set("Date", "Sat, 02 Apr 2011 04:23:52 GMT")
	cl.signRequest(r, "localhost:9000")
	c.Assert(r.Header.Get("Date"), Not(Equals), "")
	cl.signRequest(r, "localhost:9000")
	c.Assert(r.Header.Get("Authorization"), Equals, "AWS key:kHpCR/N7Rw3PwRlDd8+5X40CFVc=")
}
