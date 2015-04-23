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
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"fmt"
	"strings"
	"time"

	"net/http"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// IsValidBucketACL - is provided acl string supported
func (b BucketACL) IsValidBucketACL() bool {
	switch true {
	case b.IsPrivate():
		fallthrough
	case b.IsPublicRead():
		fallthrough
	case b.IsPublicReadWrite():
		return true
	case b.String() == "private":
		// by default its "private"
		return true
	default:
		return false
	}
}

// BucketACL - bucket level access control
type BucketACL string

// different types of ACL's currently supported for buckets
const (
	BucketPrivate         = BucketACL("private")
	BucketPublicRead      = BucketACL("public-read")
	BucketPublicReadWrite = BucketACL("public-read-write")
)

func (b BucketACL) String() string {
	if string(b) == "" {
		return "private"
	}
	return string(b)
}

// IsPrivate - is acl Private
func (b BucketACL) IsPrivate() bool {
	return b == BucketACL("private")
}

// IsPublicRead - is acl PublicRead
func (b BucketACL) IsPublicRead() bool {
	return b == BucketACL("public-read")
}

// IsPublicReadWrite - is acl PublicReadWrite
func (b BucketACL) IsPublicReadWrite() bool {
	return b == BucketACL("public-read-write")
}

/// Bucket API operations

// Get list of buckets
func (c *s3Client) listBucketsInternal() ([]*client.Item, error) {
	var res *http.Response
	var err error

	u := fmt.Sprintf("%s://%s/", c.Scheme, c.Host)
	req, err := c.getNewReq(u, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	c.signRequest(req, c.Host)

	res, err = c.Transport.RoundTrip(req)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if res != nil {
		if res.StatusCode != http.StatusOK {
			err = NewError(res)
			return nil, iodine.New(err, nil)
		}
	}
	defer res.Body.Close()

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
	if err := xml.NewDecoder(res.Body).Decode(&buckets); err != nil {
		return nil, iodine.New(client.UnexpectedError{
			Err: errors.New("Malformed response received from server")},
			map[string]string{"XMLError": err.Error()})
	}
	var items []*client.Item
	for _, b := range buckets.Buckets.Bucket {
		item := new(client.Item)
		item.Name = b.Name
		item.Time = b.CreationDate
		items = append(items, item)
	}
	return items, nil
}

// PutBucket - create new bucket
func (c *s3Client) PutBucket(acl string) error {
	bacl := BucketACL(acl)
	if !bacl.IsValidBucketACL() {
		return iodine.New(InvalidACL{ACL: acl}, nil)
	}
	bucket, _ := c.url2Object()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := c.getNewReq(u, nil)
	if err != nil {
		return iodine.New(err, nil)
	}
	req.Method = "PUT"
	// add canned ACL's while creating a bucket
	req.Header.Add("x-amz-acl", bacl.String())

	c.signRequest(req, c.Host)
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return iodine.New(err, nil)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return NewError(res)
	}

	return nil
}

// Stat - send a 'HEAD' on a bucket or object to see if exists
func (c *s3Client) Stat() error {
	bucket, _ := c.url2Object()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := c.getNewReq(u, nil)
	if err != nil {
		return iodine.New(err, nil)
	}

	req.Method = "HEAD"
	c.signRequest(req, c.Host)
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return iodine.New(err, nil)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNotFound:
		return iodine.New(BucketNotFound{Bucket: bucket}, nil)
	case http.StatusOK:
		fallthrough
	case http.StatusMovedPermanently:
		return nil
	default:
		return iodine.New(NewError(res), nil)
	}
}
