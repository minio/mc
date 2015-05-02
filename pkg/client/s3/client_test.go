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

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/minio-io/check"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func listAllMyBuckets(r io.Reader) ([]*client.Content, error) {
	type bucket struct {
		Name         string
		CreationDate time.Time
	}
	type allMyBuckets struct {
		Buckets struct {
			Bucket []*bucket
		}
	}

	var buckets allMyBuckets
	if err := xml.NewDecoder(r).Decode(&buckets); err != nil {
		return nil, iodine.New(client.UnexpectedError{Err: errors.New("Malformed response received from server")},
			map[string]string{"XMLError": err.Error()})
	}
	var contents []*client.Content
	for _, b := range buckets.Buckets.Bucket {
		content := new(client.Content)
		content.Name = b.Name
		content.Time = b.CreationDate
		contents = append(contents, content)
	}
	return contents, nil
}

func (s *MySuite) TestConfig(c *C) {
	conf := new(Config)
	conf.AccessKeyID = ""
	conf.SecretAccessKey = ""
	conf.HostURL = "http://example.com/bucket1"
	conf.UserAgent = "Minio"
	clnt := New(conf)
	c.Assert(clnt, Not(IsNil))
}

func (s *MySuite) TestBucketACL(c *C) {
	m := []struct {
		in   string
		want bool
	}{
		{"private", true},
		{"public-read", true},
		{"public-read-write", true},
		{"", false},
		{"readonly", false},
		{"invalid", false},
	}
	for _, bt := range m {
		got := isValidBucketACL(bt.in)
		c.Assert(got, Equals, bt.want)
	}
}

func (s *MySuite) TestError(c *C) {
	res := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Error><Code>AccessDenied</Code><Message>Access Denied</Message><Resource>/mybucket/myphoto.jpg</Resource><RequestId>F19772218238A85A</RequestId><HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId></Error>"
	response := new(http.Response)
	response.Body = ioutil.NopCloser(strings.NewReader(res))
	err := ResponseToError(response)
	c.Assert(err, Not(IsNil))
	c.Assert(err.Error(), Equals, "Access Denied")
}

func (s *MySuite) TestParseBuckets(c *C) {
	res := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ListAllMyBucketsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Owner><ID>ownerIDField</ID><DisplayName>bobDisplayName</DisplayName></Owner><Buckets><Bucket><Name>bucketOne</Name><CreationDate>2006-06-21T07:04:31.000Z</CreationDate></Bucket><Bucket><Name>bucketTwo</Name><CreationDate>2006-06-21T07:04:32.000Z</CreationDate></Bucket></Buckets></ListAllMyBucketsResult>"
	buckets, err := listAllMyBuckets(strings.NewReader(res))
	c.Assert(err, IsNil)
	c.Assert(len(buckets), Equals, 2)

	t1, err := time.Parse(time.RFC3339, "2006-06-21T07:04:31.000Z")
	t2, err := time.Parse(time.RFC3339, "2006-06-21T07:04:32.000Z")
	want := []*client.Content{
		{Name: "bucketOne", Time: t1},
		{Name: "bucketTwo", Time: t2},
	}
	c.Assert(buckets, DeepEquals, want)
}

func (s *MySuite) TestParseBucketsFail(c *C) {
	res := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ListAllMyBucketsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Owner><ID>ownerIDField</ID><DisplayName>bobDisplayName</DisplayName></Owner><Buckets><Bucket><Name>bucketOne</Name><CreationDate>2006-06-21T07:04:31.000Z</CreationDate></Bucket><Bucket><Name>bucketTwo</Name><CreationDate>2006-06-21T07:04:32.000Z</CreationDate></Bucket></Buckets></ListAllMyBucketsResult"
	_, err := listAllMyBuckets(strings.NewReader(res))
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestValidBucketNames(c *C) {
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
		got := client.IsValidBucketName(bt.in)
		c.Assert(got, Equals, bt.want)
	}
}
