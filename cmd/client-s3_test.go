/*
 * MinIO Client (C) 2015 MinIO, Inc.
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

package cmd

// bucketHandler is an http.Handler that verifies bucket responses and validates incoming requests
import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"

	minio "github.com/minio/minio-go/v7"
	. "gopkg.in/check.v1"
)

type bucketHandler struct {
	resource string
}

func (h bucketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		// Handler for incoming getBucketLocation request.
		if _, ok := r.URL.Query()["location"]; ok {
			response := []byte("<LocationConstraint xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"></LocationConstraint>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
			return
		}
		switch {
		case r.URL.Path == "/":
			// Handler for incoming ListBuckets request.
			response := []byte("<ListAllMyBucketsResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Buckets><Bucket><Name>bucket</Name><CreationDate>2015-05-20T23:05:09.230Z</CreationDate></Bucket></Buckets><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner></ListAllMyBucketsResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		case r.URL.Path == "/bucket/":
			// Handler for incoming ListObjects request.
			response := []byte("<ListBucketResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Contents><ETag>259d04a13802ae09c7e41be50ccc6baa</ETag><Key>object</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Delimiter></Delimiter><EncodingType></EncodingType><IsTruncated>false</IsTruncated><Marker></Marker><MaxKeys>1000</MaxKeys><Name>testbucket</Name><NextMarker></NextMarker><Prefix></Prefix></ListBucketResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		}
	case r.Method == "PUT":
		switch {
		case r.URL.Path == h.resource:
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
		// Handler for PUT object request.
		length, e := strconv.Atoi(r.Header.Get("Content-Length"))
		if e != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var buffer bytes.Buffer
		if _, e = io.CopyN(&buffer, r.Body, int64(length)); e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
	case r.Method == "HEAD":
		// Handler for Stat object request.
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(h.data)))
		w.Header().Set("Last-Modified", UTCNow().Format(http.TimeFormat))
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
	case r.Method == "POST":
		// Handler for multipart upload request.
		if _, ok := r.URL.Query()["uploads"]; ok {
			if r.URL.Path == h.resource {
				response := []byte("<InitiateMultipartUploadResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Bucket>bucket</Bucket><Key>object</Key><UploadId>EXAMPLEJZ6e0YupT2h66iePQCc9IEbYbDUy4RTpMeoSMLPRp8Z5o1u8feSRonpvnWsKKG35tI2LB9VDPiCgTy.Gq2VxQLYjrue4Nq.NBdqI-</UploadId></InitiateMultipartUploadResult>")
				w.Header().Set("Content-Length", strconv.Itoa(len(response)))
				w.Write(response)
				return
			}
		}
		if _, ok := r.URL.Query()["uploadId"]; ok {
			if r.URL.Path == h.resource {
				response := []byte("<CompleteMultipartUploadResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Location>http://bucket.s3.amazonaws.com/object</Location><Bucket>bucket</Bucket><Key>object</Key><ETag>\"3858f62230ac3c915f300c664312c11f-9\"</ETag></CompleteMultipartUploadResult>")
				w.Header().Set("Content-Length", strconv.Itoa(len(response)))
				w.Write(response)
				return
			}
		}
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	case r.Method == "GET":
		// Handler for get bucket location request.
		if _, ok := r.URL.Query()["location"]; ok {
			response := []byte("<LocationConstraint xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"></LocationConstraint>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
			return
		}
		// Handler for list multipart upload request.
		if _, ok := r.URL.Query()["uploads"]; ok {
			if r.URL.Path == "/bucket/" {
				response := []byte("<ListMultipartUploadsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Bucket>bucket</Bucket><KeyMarker/><UploadIdMarker/><NextKeyMarker/><NextUploadIdMarker/><EncodingType/><MaxUploads>1000</MaxUploads><IsTruncated>false</IsTruncated><Prefix/><Delimiter/></ListMultipartUploadsResult>")
				w.Header().Set("Content-Length", strconv.Itoa(len(response)))
				w.Write(response)
				return
			}
		}
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(h.data)))
		w.Header().Set("Last-Modified", UTCNow().Format(http.TimeFormat))
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, bytes.NewReader(h.data))
	}
}

// Test bucket operations.
func (s *TestSuite) TestBucketOperations(c *C) {
	bucket := bucketHandler(bucketHandler{
		resource: "/bucket/",
	})
	server := httptest.NewServer(bucket)
	defer server.Close()

	conf := new(Config)
	conf.HostURL = server.URL + bucket.resource
	conf.AccessKey = "WLGDGYAQYIGI833EV05A"
	conf.SecretKey = "BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"
	conf.Signature = "S3v4"
	s3c, err := S3New(conf)
	c.Assert(err, IsNil)

	err = s3c.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, IsNil)

	conf.HostURL = server.URL + string(s3c.GetURL().Separator)
	s3c, err = S3New(conf)
	c.Assert(err, IsNil)

	for content := range s3c.List(globalContext, ListOptions{ShowDir: DirNone}) {
		c.Assert(content.Err, IsNil)
		c.Assert(content.Type.IsDir(), Equals, true)
	}

	conf.HostURL = server.URL + "/bucket"
	s3c, err = S3New(conf)
	c.Assert(err, IsNil)

	for content := range s3c.List(globalContext, ListOptions{ShowDir: DirNone}) {
		c.Assert(content.Err, IsNil)
		c.Assert(content.Type.IsDir(), Equals, true)
	}

	conf.HostURL = server.URL + "/bucket/"
	s3c, err = S3New(conf)
	c.Assert(err, IsNil)

	for content := range s3c.List(globalContext, ListOptions{ShowDir: DirNone}) {
		c.Assert(content.Err, IsNil)
		c.Assert(content.Type.IsRegular(), Equals, true)
	}
}

// Test all object operations.
func (s *TestSuite) TestObjectOperations(c *C) {
	object := objectHandler(objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	})
	server := httptest.NewServer(object)
	defer server.Close()

	conf := new(Config)
	conf.HostURL = server.URL + object.resource
	conf.AccessKey = "WLGDGYAQYIGI833EV05A"
	conf.SecretKey = "BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"
	conf.Signature = "S3v4"
	s3c, err := S3New(conf)
	c.Assert(err, IsNil)

	var reader io.Reader
	reader = bytes.NewReader(object.data)
	n, err := s3c.Put(context.Background(), reader, int64(len(object.data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(object.data)))

	reader, err = s3c.Get(context.Background(), GetOptions{})
	c.Assert(err, IsNil)
	var buffer bytes.Buffer
	{
		_, err := io.Copy(&buffer, reader)
		c.Assert(err, IsNil)
		c.Assert(buffer.Bytes(), DeepEquals, object.data)
	}
}

var testSelectCompressionTypeCases = []struct {
	opts            SelectObjectOpts
	object          string
	compressionType minio.SelectCompressionType
}{
	{SelectObjectOpts{CompressionType: minio.SelectCompressionNONE}, "a.gzip", minio.SelectCompressionNONE},
	{SelectObjectOpts{CompressionType: minio.SelectCompressionBZIP}, "a.gz", minio.SelectCompressionBZIP},
	{SelectObjectOpts{}, "t.parquet", minio.SelectCompressionNONE},
	{SelectObjectOpts{}, "x.csv.gz", minio.SelectCompressionGZIP},
	{SelectObjectOpts{}, "x.json.bz2", minio.SelectCompressionBZIP},
	{SelectObjectOpts{}, "b.gz", minio.SelectCompressionGZIP},
	{SelectObjectOpts{}, "k.bz2", minio.SelectCompressionBZIP},
	{SelectObjectOpts{}, "a.csv", minio.SelectCompressionNONE},
	{SelectObjectOpts{}, "a.json", minio.SelectCompressionNONE},
}

// TestSelectCompressionType - tests compression type returned
// by method
func (s *TestSuite) TestSelectCompressionType(c *C) {
	for _, test := range testSelectCompressionTypeCases {
		cType := selectCompressionType(test.opts, test.object)
		c.Assert(cType, DeepEquals, test.compressionType)
	}
}
