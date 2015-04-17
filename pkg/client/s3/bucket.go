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
	"fmt"
	"strings"

	"net/http"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

/// Bucket API operations

// ListBuckets - Get list of buckets
func (c *s3Client) ListBuckets() ([]*client.Bucket, error) {
	var res *http.Response
	var err error

	u := fmt.Sprintf("%s://%s/", c.Scheme, c.Host)
	req, err := newReq(u, c.UserAgent, nil)
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

	return listAllMyBuckets(res.Body)
}

// PutBucket - create new bucket
func (c *s3Client) PutBucket(bucket string) error {
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := newReq(u, c.UserAgent, nil)
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

func (c *s3Client) StatBucket(bucket string) error {
	if bucket == "" {
		return iodine.New(client.InvalidArgument{}, nil)
	}
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	u := fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	req, err := newReq(u, c.UserAgent, nil)
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

func (c *s3Client) ListObjects(bucket, objectPrefix string) (items []*client.Item, err error) {
	item, err := c.GetObjectMetadata(bucket, objectPrefix)
	switch err {
	case nil: // List a single object. Exact key
		items = append(items, item)
		return items, nil
	default:
		// List all objects matching the key prefix
		items, _, err = c.queryObjects(bucket, "", objectPrefix, "", globalMaxKeys)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		// even if items are equal to '0' is valid case
		return items, nil
	}
}
