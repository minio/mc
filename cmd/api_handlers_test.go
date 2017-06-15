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

package cmd

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type objectAPIHandler struct {
	lock   *sync.Mutex
	bucket string
	object map[string][]byte
}

func (h objectAPIHandler) getHandler(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()
	switch {
	case r.URL.Path == "/":
		response := []byte("<ListAllMyBucketsResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Buckets><Bucket><Name>bucket</Name><CreationDate>2015-05-20T23:05:09.230Z</CreationDate></Bucket></Buckets><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner></ListAllMyBucketsResult>")
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		w.Write(response)
		return
	case r.URL.Path == "/bucket":
		_, ok := r.URL.Query()["acl"]
		if ok {
			response := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><AccessControlPolicy><Owner><ID>75aa57f09aa0c8caeab4f8c24e99d10f8e7faeebf76c078efc7c6caea54ba06a</ID><DisplayName>CustomersName@amazon.com</DisplayName></Owner><AccessControlList><Grant><Grantee xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"CanonicalUser\"><ID>75aa57f09aa0c8caeab4f8c24e99d10f8e7faeebf76c078efc7c6caea54ba06a</ID><DisplayName>CustomersName@amazon.com</DisplayName></Grantee><Permission>FULL_CONTROL</Permission></Grant></AccessControlList></AccessControlPolicy>")
			w.Header().Set("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
			return
		}
		fallthrough
	case r.URL.Path == "/bucket":
		response := []byte("<ListBucketResult xmlns=\"http://doc.s3.amazonaws.com/2006-03-01\"><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object0</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object1</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object2</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object3</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object4</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object5</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object6</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Contents><ETag>b1946ac92492d2347c6235b4d2611184</ETag><Key>object7</Key><LastModified>2015-05-21T18:24:21.097Z</LastModified><Size>22061</Size><Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner><StorageClass>STANDARD</StorageClass></Contents><Delimiter></Delimiter><EncodingType></EncodingType><IsTruncated>false</IsTruncated><Marker></Marker><MaxKeys>1000</MaxKeys><Name>testbucket</Name><NextMarker></NextMarker><Prefix></Prefix></ListBucketResult>")
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		w.Write(response)
		return
	case r.URL.Path != "":
		if _, ok := h.object[filepath.Base(r.URL.Path)]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(h.object[filepath.Base(r.URL.Path)])))
		w.Header().Set("Last-Modified", UTCNow().Format(http.TimeFormat))
		w.Header().Set("ETag", "b1946ac92492d2347c6235b4d2611184")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, bytes.NewReader(h.object[filepath.Base(r.URL.Path)]))
		return
	}
}

func (h objectAPIHandler) headHandler(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()
	switch {
	case r.URL.Path == "/":
		w.WriteHeader(http.StatusOK)
		return
	case r.URL.Path == "/bucket":
		w.WriteHeader(http.StatusOK)
		return
	case r.URL.Path != "":
		w.Header().Set("Content-Length", strconv.Itoa(len(h.object[filepath.Base(r.URL.Path)])))
		w.Header().Set("Last-Modified", UTCNow().Format(http.TimeFormat))
		w.Header().Set("ETag", "b1946ac92492d2347c6235b4d2611184")
		w.WriteHeader(http.StatusOK)
		return
	}
}

func isValidBucket(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if bucket[0] == '.' || bucket[len(bucket)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucket); match {
		return false
	}
	// We don't support buckets with '.' in them
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", bucket)
	return match
}

func (h objectAPIHandler) putHandler(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()
	switch {
	case r.URL.Path == "/":
		w.WriteHeader(http.StatusBadRequest)
		return
	case r.URL.Path == "/bucket":
		_, ok := r.URL.Query()["acl"]
		if ok {
			switch r.Header.Get("x-amz-acl") {
			case "public-read-write":
				fallthrough
			case "public-read":
				fallthrough
			case "private":
				fallthrough
			case "authenticated-read":
				w.WriteHeader(http.StatusOK)
				return
			default:
				w.WriteHeader(http.StatusNotImplemented)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	case r.URL.Path != "":
		if !isValidBucket(strings.Split(r.URL.Path, "/")[1]) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

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
		h.object[filepath.Base(r.URL.Path)] = buffer.Bytes()
		w.Header().Set("ETag", "b1946ac92492d2347c6235b4d2611184")
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (h objectAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		h.getHandler(w, r)
	case r.Method == "HEAD":
		h.headHandler(w, r)
	case r.Method == "PUT":
		h.putHandler(w, r)
	}
}
