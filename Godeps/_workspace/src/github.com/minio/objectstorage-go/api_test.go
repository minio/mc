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

package objectstorage_test

import (
	"net/http/httptest"
	"testing"

	. "github.com/minio/objectstorage-go"
)

func TestBucketOperations(t *testing.T) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()
}

func TestObjectOperations(t *testing.T) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()
}

func TestPartSize(t *testing.T) {
	var maxPartSize uint64 = 1024 * 1024 * 1024 * 5
	partSize := GetPartSize(5000000000000000000)
	if partSize > MinimumPartSize {
		if partSize != maxPartSize {
			t.Fatal("invalid result, cannot be bigger than MaxPartSize 5GB")
		}
	}
	partSize = GetPartSize(50000000000)
	if partSize > MinimumPartSize {
		t.Fatal("invalid result, cannot be bigger than MinimumPartSize 5MB")
	}
}
