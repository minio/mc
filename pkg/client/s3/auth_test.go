/*
 * Mini Copy (C) 2015 Minio, Inc.
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

func (s *MySuite) TestStringToSign(c *C) {
	var a s3Client
	tests := []reqAndExpected{
		{`GET /photos/puppy.jpg HTTP/1.1
Host: johnsmith.s3.amazonaws.com
Date: Tue, 27 Mar 2007 19:36:42 +0000

`,
			"GET\n\n\nTue, 27 Mar 2007 19:36:42 +0000\n/johnsmith/photos/puppy.jpg", "s3.amazonaws.com"},
		{`PUT /photos/puppy.jpg HTTP/1.1
Content-Type: image/jpeg
Content-Length: 94328
Host: johnsmith.s3.amazonaws.com
Date: Tue, 27 Mar 2007 21:15:45 +0000

`,
			"PUT\n\nimage/jpeg\nTue, 27 Mar 2007 21:15:45 +0000\n/johnsmith/photos/puppy.jpg", "s3.amazonaws.com"},
		{`GET /?prefix=photos&max-keys=50&marker=puppy HTTP/1.1
User-Agent: Mozilla/5.0
Host: johnsmith.s3.amazonaws.com
Date: Tue, 27 Mar 2007 19:42:41 +0000

`,
			"GET\n\n\nTue, 27 Mar 2007 19:42:41 +0000\n/johnsmith/", "s3.amazonaws.com"},
		{`DELETE /johnsmith/photos/puppy.jpg HTTP/1.1
User-Agent: dotnet
Host: s3.amazonaws.com
Date: Tue, 27 Mar 2007 21:20:27 +0000
x-amz-date: Tue, 27 Mar 2007 21:20:26 +0000

`,
			"DELETE\n\n\n\nx-amz-date:Tue, 27 Mar 2007 21:20:26 +0000\n/johnsmith/photos/puppy.jpg", "s3.amazonaws.com"},
		{`PUT /db-backup.dat.gz HTTP/1.1
User-Agent: curl/7.15.5
Host: static.johnsmith.net:8080
Date: Tue, 27 Mar 2007 21:06:08 +0000
x-amz-acl: public-read
content-type: application/x-download
Content-MD5: 4gJE4saaMU4BqNR0kLY+lw==
X-Amz-Meta-ReviewedBy: joe@johnsmith.net
X-Amz-Meta-ReviewedBy: jane@johnsmith.net
X-Amz-Meta-FileChecksum: 0x02661779
X-Amz-Meta-ChecksumAlgorithm: crc32
Content-Disposition: attachment; filename=database.dat
Content-Encoding: gzip
Content-Length: 5913339

`,
			"PUT\n4gJE4saaMU4BqNR0kLY+lw==\napplication/x-download\nTue, 27 Mar 2007 21:06:08 +0000\nx-amz-acl:public-read\nx-amz-meta-checksumalgorithm:crc32\nx-amz-meta-filechecksum:0x02661779\nx-amz-meta-reviewedby:joe@johnsmith.net,jane@johnsmith.net\n/db-backup.dat.gz", "static.johnsmith.net:8080"},
	}
	for _, test := range tests {
		got := a.stringToSign(req(test.req), test.host)
		c.Assert(got, Equals, test.expected)
	}
}

func (s *MySuite) TestBucketFromHostname(c *C) {
	var a s3Client
	tests := []reqAndExpected{
		{"GET / HTTP/1.0\n\n", "", ""},
		{"GET / HTTP/1.0\nHost: s3.amazonaws.com\n\n", "", "s3.amazonaws.com"},
		{"GET / HTTP/1.0\nHost: foo.s3.amazonaws.com\n\n", "foo", "s3.amazonaws.com"},
		{"GET / HTTP/1.0\nHost: foo.com:123\n\n", "foo.com", "foo.com"},
		{"GET / HTTP/1.0\nHost: bar.com\n\n", "", "bar.com"},
	}
	for _, test := range tests {
		got := a.bucketFromHost(req(test.req), test.host)
		c.Assert(got, Equals, test.expected)
	}
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
	cl.signRequest(r, "localhost:9000")
	c.Assert(r.Header.Get("Date"), Not(Equals), "")
	r.Header.Set("Date", "Sat, 02 Apr 2011 04:23:52 GMT")
	cl.signRequest(r, "localhost:9000")
	c.Assert(r.Header.Get("Authorization"), Equals, "AWS key:kHpCR/N7Rw3PwRlDd8+5X40CFVc=")
}

func (s *MySuite) TestHasDotSuffix(c *C) {
	c.Assert(hasDotSuffix("foo.com", "com"), Equals, true)
	c.Assert(hasDotSuffix("foocom", "com"), Equals, false)
	c.Assert(hasDotSuffix("com", "com"), Equals, false)
}
