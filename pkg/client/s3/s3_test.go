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
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/minio/mc/pkg/client"

	. "gopkg.in/check.v1"
)

type bucketHandler struct {
	resource string
}

func (h bucketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		switch {
		case r.URL.Path == "/":
			response := []byte("<ListAllMyBucketsResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Buckets><Bucket><Name>bucket</Name><CreationDate>2015-05-20T23:05:09.230Z</CreationDate></Bucket></Buckets><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner></ListAllMyBucketsResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		case r.URL.Path == "/bucket":
			response := []byte("<ListBucketResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Contents><ETag>259d04a13802ae09c7e41be50ccc6baa</ETag><Key>object</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Delimiter></Delimiter><EncodingType></EncodingType><IsTruncated>false</IsTruncated><Marker></Marker><MaxKeys>1000</MaxKeys><Name>testbucket</Name><NextMarker></NextMarker><Prefix></Prefix></ListBucketResult>")
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

// objectHandler is an http.Handler that verifies object responses and validates incoming requests
type objectHandler struct {
	resource string
	data     []byte
}

func (h objectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "PUT":
		length, err := strconv.Atoi(r.Header.Get("Content-Length"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var buffer bytes.Buffer
		_, err = io.CopyN(&buffer, r.Body, int64(length))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !bytes.Equal(h.data, buffer.Bytes()) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
	case r.Method == "HEAD":
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(h.data)))
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
	case r.Method == "GET":
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(h.data)))
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, bytes.NewReader(h.data))
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

	conf := new(client.Config)
	conf.HostURL = server.URL + bucket.resource
	s3c, err := New(conf)
	c.Assert(err, IsNil)

	err = s3c.MakeBucket()
	c.Assert(err, IsNil)

	err = s3c.SetBucketAccess("public-read-write")
	c.Assert(err, IsNil)

	conf.HostURL = server.URL + string(s3c.GetURL().Separator)
	s3c, err = New(conf)
	c.Assert(err, IsNil)

	for content := range s3c.List(false, false) {
		c.Assert(content.Err, IsNil)
		c.Assert(content.Content.Type.IsDir(), Equals, true)
	}

	conf.HostURL = server.URL + "/bucket"
	s3c, err = New(conf)
	c.Assert(err, IsNil)

	for content := range s3c.List(false, false) {
		c.Assert(content.Err, IsNil)
		c.Assert(content.Content.Type.IsRegular(), Equals, true)
	}
}

func (s *MySuite) TestObjectOperations(c *C) {
	object := objectHandler(objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	})
	server := httptest.NewServer(object)
	defer server.Close()

	conf := new(client.Config)
	conf.HostURL = server.URL + object.resource
	s3c, err := New(conf)
	c.Assert(err, IsNil)

	err = s3c.Put(int64(len(object.data)), bytes.NewReader(object.data))
	c.Assert(err, IsNil)

	content, err := s3c.Stat()
	c.Assert(err, IsNil)
	c.Assert(content.Size, Equals, int64(len(object.data)))
	c.Assert(content.Type.IsRegular(), Equals, true)

	reader, size, err := s3c.Get(0, 0)
	c.Assert(size, Equals, int64(len(object.data)))

	var buffer bytes.Buffer
	{
		_, err := io.CopyN(&buffer, reader, int64(size))
		c.Assert(err, IsNil)
		c.Assert(buffer.Bytes(), DeepEquals, object.data)
	}
}
