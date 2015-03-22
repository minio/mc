// Original license //
// ---------------- //

/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// All other modifications and improvements //
// ---------------------------------------- //

/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"reflect"
	"strings"
	"testing"
	"time"
)

var tc *Client

func TestParseBuckets(t *testing.T) {
	res := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ListAllMyBucketsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Owner><ID>ownerIDField</ID><DisplayName>bobDisplayName</DisplayName></Owner><Buckets><Bucket><Name>bucketOne</Name><CreationDate>2006-06-21T07:04:31.000Z</CreationDate></Bucket><Bucket><Name>bucketTwo</Name><CreationDate>2006-06-21T07:04:32.000Z</CreationDate></Bucket></Buckets></ListAllMyBucketsResult>"
	buckets, err := parseListAllMyBuckets(strings.NewReader(res))
	if err != nil {
		t.Fatal(err)
	}
	if g, w := len(buckets), 2; g != w {
		t.Errorf("num parsed buckets = %d; want %d", g, w)
	}

	t1, err := time.Parse(iso8601Format, "2006-06-21T07:04:31.000Z")
	t2, err := time.Parse(iso8601Format, "2006-06-21T07:04:32.000Z")
	want := []*Bucket{
		{Name: "bucketOne", CreationDate: xmlTime{t1}},
		{Name: "bucketTwo", CreationDate: xmlTime{t2}},
	}
	dump := func(v []*Bucket) {
		for i, b := range v {
			t.Logf("Bucket #%d: %#v", i, b)
		}
	}
	if !reflect.DeepEqual(buckets, want) {
		t.Error("mismatch; GOT:")
		dump(buckets)
		t.Error("WANT:")
		dump(want)
	}
}

func TestValidBucketNames(t *testing.T) {
	m := []struct {
		in   string
		want bool
	}{
		{"myawsbucket", true},
		{"myaws-bucket", true},
		{"my-aws-bucket", true},
		{"my.aws.bucket", false},
		{"my-aws-bucket.1", false},
		{"my---bucket.1", false},
		{".myawsbucket", false},
		{"-myawsbucket", false},
		{"myawsbucket.", false},
		{"myawsbucket-", false},
		{"my..awsbucket", false},
	}

	for _, bt := range m {
		got := IsValidBucketName(bt.in)
		if got != bt.want {
			t.Errorf("func(%q) = %v; want %v", bt.in, got, bt.want)
		}
	}
}
