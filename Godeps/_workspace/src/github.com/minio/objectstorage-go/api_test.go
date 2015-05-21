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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
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
		if r.URL.Path == "/" {
			response := []byte("<ListAllMyBucketsResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Buckets><Bucket><Name>bucket</Name><CreationDate>2015-05-20T23:05:09.230Z</CreationDate></Bucket></Buckets><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner></ListAllMyBucketsResult>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		}
	case r.Method == "POST":
		w.WriteHeader(http.StatusOK)
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
	case r.Method == "HEAD":
	case r.Method == "POST":
	case r.Method == "GET":
	case r.Method == "DELETE":
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
