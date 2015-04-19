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

/// Bucket API operations

// ListBuckets - Get list of buckets
func (c *s3Client) listBuckets() ([]*client.Item, error) {
	var res *http.Response
	var err error

	u := fmt.Sprintf("%s://%s/", c.Scheme, c.Host)
	req, err := getNewReq(u, c.UserAgent, nil)
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
		return nil, iodine.New(client.UnexpectedError{Err: errors.New("Malformed response received from server")},
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
func (c *s3Client) PutBucket() error {
	bucket, _ := c.url2Object()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := getNewReq(u, c.UserAgent, nil)
	if err != nil {
		return iodine.New(err, nil)
	}

	req.Method = "PUT"
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

// StatBucket - send a 'HEAD' on a bucket to see if exists or not
func (c *s3Client) StatBucket() error {
	bucket, _ := c.url2Object()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := getNewReq(u, c.UserAgent, nil)
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
		return iodine.New(client.BucketNotFound{Bucket: bucket}, nil)
	case http.StatusOK:
		fallthrough
	case http.StatusMovedPermanently:
		return nil
	default:
		return iodine.New(NewError(res), nil)
	}
}
