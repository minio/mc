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

package cmd

// bucketHandler is an http.Handler that verifies bucket responses and validates incoming requests
import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/minio/minio-go/v7"
	checkv1 "gopkg.in/check.v1"
)

type bucketHandler struct {
	resource string
}

func (h bucketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Handler for incoming getBucketLocation request.
		if _, ok := r.URL.Query()["location"]; ok {
			response := []byte("<LocationConstraint xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"></LocationConstraint>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
			return
		}
		switch r.URL.Path {
		case "/":
			// Handler for incoming ListBuckets request.
			response := []byte("<ListAllMyBucketsResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Buckets><Bucket><Name>bucket</Name><CreationDate>2015-05-20T23:05:09.230Z</CreationDate></Bucket></Buckets><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner></ListAllMyBucketsResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		case "/bucket/":
			// Handler for incoming ListObjects request.
			response := []byte("<ListBucketResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Contents><ETag>259d04a13802ae09c7e41be50ccc6baa</ETag><Key>object</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Delimiter></Delimiter><EncodingType></EncodingType><IsTruncated>false</IsTruncated><Marker></Marker><MaxKeys>1000</MaxKeys><Name>testbucket</Name><NextMarker></NextMarker><Prefix></Prefix></ListBucketResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		}
	case "PUT":
		switch r.URL.Path {
		case h.resource:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	case "HEAD":
		switch r.URL.Path {
		case h.resource:
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
	if ak := r.Header.Get("Authorization"); len(ak) == 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPut:
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
	case http.MethodHead:
		// Handler for Stat object request.
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(h.data)))
		w.Header().Set("Last-Modified", UTCNow().Format(http.TimeFormat))
		w.Header().Set("ETag", "9af2f8218b150c351ad802c6f3d66abe")
		w.WriteHeader(http.StatusOK)
	case http.MethodPost:
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
	case http.MethodGet:
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

type stsHandler struct {
	endpoint string
	jwt      []byte
}

func (h stsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := ParseForm(r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodPost:
		token := r.Form.Get("WebIdentityToken")
		if token == string(h.jwt) {
			response := []byte("<AssumeRoleWithWebIdentityResponse xmlns=\"https://sts.amazonaws.com/doc/2011-06-15/\"><AssumeRoleWithWebIdentityResult><AssumedRoleUser><Arn></Arn><AssumeRoleId></AssumeRoleId></AssumedRoleUser><Credentials><AccessKeyId>7NL5BR739GUQ0ZOD4JNB</AccessKeyId><SecretAccessKey>A2mxZSxPnHNhSduedUHczsXZpVSSssOLpDruUmTV</SecretAccessKey><Expiration>0001-01-01T00:00:00Z</Expiration><SessionToken>eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiI3Tkw1QlI3MzlHVVEwWk9ENEpOQiIsImV4cCI6MTY5OTYwMzMwNiwicGFyZW50IjoibWluaW8iLCJzZXNzaW9uUG9saWN5IjoiZXlKV1pYSnphVzl1SWpvaU1qQXhNaTB4TUMweE55SXNJbE4wWVhSbGJXVnVkQ0k2VzNzaVJXWm1aV04wSWpvaVFXeHNiM2NpTENKQlkzUnBiMjRpT2xzaVlXUnRhVzQ2S2lKZGZTeDdJa1ZtWm1WamRDSTZJa0ZzYkc5M0lpd2lRV04wYVc5dUlqcGJJbXR0Y3pvcUlsMTlMSHNpUldabVpXTjBJam9pUVd4c2IzY2lMQ0pCWTNScGIyNGlPbHNpY3pNNktpSmRMQ0pTWlhOdmRYSmpaU0k2V3lKaGNtNDZZWGR6T25Nek9qbzZLaUpkZlYxOSJ9.uuE_x7PO8QoPfUk9KzUELoAqxihIknZAvJLl5aYJjwpSjJYFTPLp6EvuyJX2hc18s9HzeiJ-vU0dPzsy50dXmg</SessionToken></Credentials></AssumeRoleWithWebIdentityResult><ResponseMetadata></ResponseMetadata></AssumeRoleWithWebIdentityResponse>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Header().Set("Content-Type", "application/xml")
			w.Header().Set("Server", "MinIO")
			w.Write(response)
			return
		}
		response := []byte("<ErrorResponse xmlns=\"https://sts.amazonaws.com/doc/2011-06-15/\"><Error><Type></Type><Code>AccessDenied</Code><Message>Access denied: Invalid Token</Message></Error><RequestId></RequestId></ErrorResponse>")
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		w.Header().Set("Content-Type", "application/xml")
		w.Write(response)
		return
	}
}

// Test bucket operations.
func (s *TestSuite) TestBucketOperations(c *checkv1.C) {
	bucket := bucketHandler{
		resource: "/bucket/",
	}
	server := httptest.NewServer(bucket)
	defer server.Close()

	conf := new(Config)
	conf.HostURL = server.URL + bucket.resource
	conf.AccessKey = "WLGDGYAQYIGI833EV05A"
	conf.SecretKey = "BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"
	conf.Signature = "S3v4"
	s3c, err := S3New(conf)
	c.Assert(err, checkv1.IsNil)

	err = s3c.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, checkv1.IsNil)

	conf.HostURL = server.URL + string(s3c.GetURL().Separator)
	s3c, err = S3New(conf)
	c.Assert(err, checkv1.IsNil)

	for content := range s3c.List(globalContext, ListOptions{ShowDir: DirNone}) {
		c.Assert(content.Err, checkv1.IsNil)
		c.Assert(content.Type.IsDir(), checkv1.Equals, true)
	}

	conf.HostURL = server.URL + "/bucket"
	s3c, err = S3New(conf)
	c.Assert(err, checkv1.IsNil)

	for content := range s3c.List(globalContext, ListOptions{ShowDir: DirNone}) {
		c.Assert(content.Err, checkv1.IsNil)
		c.Assert(content.Type.IsDir(), checkv1.Equals, true)
	}

	conf.HostURL = server.URL + "/bucket/"
	s3c, err = S3New(conf)
	c.Assert(err, checkv1.IsNil)

	for content := range s3c.List(globalContext, ListOptions{ShowDir: DirNone}) {
		c.Assert(content.Err, checkv1.IsNil)
		c.Assert(content.Type.IsRegular(), checkv1.Equals, true)
	}
}

// Test all object operations.
func (s *TestSuite) TestObjectOperations(c *checkv1.C) {
	object := objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	}
	server := httptest.NewServer(object)
	defer server.Close()

	conf := new(Config)
	conf.HostURL = server.URL + object.resource
	conf.AccessKey = "WLGDGYAQYIGI833EV05A"
	conf.SecretKey = "BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"
	conf.Signature = "S3v4"
	s3c, err := S3New(conf)
	c.Assert(err, checkv1.IsNil)

	var reader io.Reader
	reader = bytes.NewReader(object.data)
	n, err := s3c.Put(context.Background(), reader, int64(len(object.data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	c.Assert(err, checkv1.IsNil)
	c.Assert(n, checkv1.Equals, int64(len(object.data)))

	reader, _, err = s3c.Get(context.Background(), GetOptions{})
	c.Assert(err, checkv1.IsNil)
	var buffer bytes.Buffer
	{
		_, err := io.Copy(&buffer, reader)
		c.Assert(err, checkv1.IsNil)
		c.Assert(buffer.Bytes(), checkv1.DeepEquals, object.data)
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
func (s *TestSuite) TestSelectCompressionType(c *checkv1.C) {
	for _, test := range testSelectCompressionTypeCases {
		cType := selectCompressionType(test.opts, test.object)
		c.Assert(cType, checkv1.DeepEquals, test.compressionType)
	}
}
