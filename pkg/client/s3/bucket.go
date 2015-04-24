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
 * Mini Copy (C) 2015 Minio, Inc.
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

func isValidBucketACL(acl string) bool {
	switch acl {
	case "private":
		fallthrough
	case "public-read":
		fallthrough
	case "public-read-write":
		fallthrough
	case "":
		return true
	default:
		return false
	}
}

/// Bucket API operations

// Get list of buckets
func (c *s3Client) listBucketsInternal() ([]*client.Item, error) {
	var res *http.Response
	var err error

	u := fmt.Sprintf("%s://%s/", c.Scheme, c.Host)
	req, err := c.newRequest("GET", u, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}

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
	if !isValidBucketACL(acl) {
		return iodine.New(InvalidACL{ACL: acl}, nil)
	}
	if acl == "" {
		acl = "private"
	}
	bucket, _ := c.url2BucketAndObject()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	// new request
	req, err := c.newRequest("PUT", u, nil)
	if err != nil {
		return iodine.New(err, nil)
	}
	// add canned ACL's while creating a bucket
	req.Header.Add("x-amz-acl", acl)

	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return iodine.New(err, nil)
	}
	if res != nil {
		if res.StatusCode != http.StatusOK {
			return iodine.New(NewError(res), nil)
		}
	}
	defer res.Body.Close()
	return nil
}

// Stat - send a 'HEAD' on a bucket or object to see if exists
func (c *s3Client) Stat() error {
	bucket, _ := c.url2BucketAndObject()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := c.newRequest("HEAD", u, nil)
	if err != nil {
		return iodine.New(err, nil)
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return iodine.New(err, nil)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		fallthrough
	case http.StatusMovedPermanently:
		return nil
	default:
		return iodine.New(NewError(res), nil)
	}
}
