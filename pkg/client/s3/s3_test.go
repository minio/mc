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

// bucketHandler is an http.Handler that verifies bucket responses and validates incoming requests
import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	. "github.com/minio/check"
)

type bucketHandler struct {
	resource string
}

func (h bucketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		if r.URL.Path == "/" {
			response := []byte("<ListAllMyBucketsResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Buckets><Bucket><Name>bucket</Name><CreationDate>2015-05-20T23:05:09.230Z</CreationDate></Bucket></Buckets><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner></ListAllMyBucketsResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		}
	case r.Method == "PUT":
		switch {
		case r.URL.Path == h.resource:
			_, ok := r.URL.Query()["acl"]
			if ok {
				if r.Header.Get("x-amz-acl") != "public-read-write" {
					w.WriteHeader(http.StatusNotImplemented)
				}
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	case r.Method == "HEAD":
		switch {
		case r.URL.Path == h.resource:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}
}

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestBucketOperations(c *C) {
	bucket := bucketHandler(bucketHandler{
		resource: "/bucket",
	})
	server := httptest.NewServer(bucket)
	defer server.Close()

	conf := new(Config)
	conf.HostURL = server.URL + bucket.resource
	s3c, err := New(conf)
	c.Assert(err, IsNil)

	err = s3c.MakeBucket()
	c.Assert(err, IsNil)

	err = s3c.SetBucketACL("public-read-write")
	c.Assert(err, IsNil)

	conf.HostURL = server.URL + "/"
	s3c, err = New(conf)
	c.Assert(err, IsNil)

	for content := range s3c.List(false) {
		c.Assert(content.Err, IsNil)
		c.Assert(content.Content.Name, Equals, "bucket")
		c.Assert(content.Content.Type.IsDir(), Equals, true)
	}
}
