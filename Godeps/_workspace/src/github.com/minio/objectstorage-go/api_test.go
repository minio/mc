/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package objectstorage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func ExampleGetPartSize() {
	fmt.Println(GetPartSize(5000000000))
	// Output: 5242880
}
func ExampleGetPartSize_second() {
	fmt.Println(GetPartSize(50000000000000000))
	// Output: 5368709120
}

// bucketHandler is an http.Handler that verifies bucket responses and validates incoming requests
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
	case r.Method == "DELETE":
		switch {
		case r.URL.Path != h.resource:
			w.WriteHeader(http.StatusNotFound)
		default:
			h.resource = ""
			w.WriteHeader(http.StatusOK)
		}
	}
}

func TestBucketOperations(t *testing.T) {
	bucket := bucketHandler(bucketHandler{
		resource: "/bucket",
	})
	server := httptest.NewServer(bucket)
	defer server.Close()

	a := New(&Config{Endpoint: server.URL})
	err := a.MakeBucket("bucket", "private", "")
	if err != nil {
		t.Errorf("Error")
	}

	err = a.StatBucket("bucket")
	if err != nil {
		t.Errorf("Error")
	}

	err = a.StatBucket("bucket1")
	if err == nil {
		t.Errorf("Error")
	}
	if err.Error() != "403 Forbidden" {
		t.Errorf("Error")
	}

	err = a.SetBucketACL("bucket", "public-read-write")
	if err != nil {
		t.Errorf("Error")
	}

	for b := range a.ListBuckets() {
		if b.Err != nil {
			t.Fatalf(b.Err.Error())
		}
		if b.Data.Name != "bucket" {
			t.Errorf("Error")
		}
	}

	for o := range a.ListObjects("bucket", "", true) {
		if o.Err != nil {
			t.Fatalf(o.Err.Error())
		}
		if o.Data.Key != "object" {
			t.Errorf("Error")
		}
	}

	err = a.DeleteBucket("bucket")
	if err != nil {
		t.Errorf("Error")
	}

	err = a.DeleteBucket("bucket1")
	if err == nil {
		t.Fatalf("Error")
	}
	if err.Error() != "404 Not Found" {
		t.Errorf("Error")
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
	case r.Method == "POST":
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
	case r.Method == "DELETE":
		if r.URL.Path != h.resource {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.resource = ""
		h.data = nil
		w.WriteHeader(http.StatusOK)
	}
}

func TestObjectOperations(t *testing.T) {
	object := objectHandler(objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	})
	server := httptest.NewServer(object)
	defer server.Close()

	a := New(&Config{Endpoint: server.URL})
	data := []byte("Hello, World")
	err := a.PutObject("bucket", "object", uint64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Error")
	}
	metadata, err := a.StatObject("bucket", "object")
	if err != nil {
		t.Fatalf("Error")
	}
	if metadata.Key != "object" {
		t.Fatalf("Error")
	}
	if metadata.ETag != "9af2f8218b150c351ad802c6f3d66abe" {
		t.Fatalf("Error")
	}

	reader, metadata, err := a.GetObject("bucket", "object", 0, 0)
	if err != nil {
		t.Fatalf("Error")
	}
	if metadata.Key != "object" {
		t.Fatalf("Error")
	}
	if metadata.ETag != "9af2f8218b150c351ad802c6f3d66abe" {
		t.Fatalf("Error")
	}

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, reader)
	if !bytes.Equal(buffer.Bytes(), data) {
		t.Fatalf("Error")
	}

	err = a.DeleteObject("bucket", "object")
	if err != nil {
		t.Fatalf("Error")
	}
}

func TestPartSize(t *testing.T) {
	var maxPartSize uint64 = 1024 * 1024 * 1024 * 5
	partSize := GetPartSize(5000000000000000000)
	if partSize > MinimumPartSize {
		if partSize > maxPartSize {
			t.Fatal("invalid result, cannot be bigger than MaxPartSize 5GB")
		}
	}
	partSize = GetPartSize(50000000000)
	if partSize > MinimumPartSize {
		t.Fatal("invalid result, cannot be bigger than MinimumPartSize 5MB")
	}
}

func TestURLEncoding(t *testing.T) {
	type urlStrings struct {
		name        string
		encodedName string
	}

	want := []urlStrings{
		{
			name:        "bigfile-1._%",
			encodedName: "bigfile-1._%25",
		},
		{
			name:        "本語",
			encodedName: "%E6%9C%AC%E8%AA%9E",
		},
		{
			name:        "本b語.1",
			encodedName: "%E6%9C%ACb%E8%AA%9E.1",
		},
		{
			name:        ">123>3123123",
			encodedName: "%3E123%3E3123123",
		},
	}

	for _, u := range want {
		encodedName, err := urlEncodeName(u.name)
		if err != nil {
			t.Fatalf("Error")
		}
		if u.encodedName != encodedName {
			t.Errorf("Error")
		}
	}
}
