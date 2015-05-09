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
	"fmt"

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
	case "authenticated-read":
		return true
	default:
		return false
	}
}

/// Bucket API operations

// PutBucket - create a new bucket
func (c *s3Client) PutBucket() error {
	_, object := c.url2BucketAndObject()
	if object != "" {
		return iodine.New(InvalidQueryURL{URL: ""}, nil)
	}
	requestURL, err := c.getRequestURL()
	if err != nil {
		return iodine.New(err, nil)
	}
	// new request
	req, err := c.newRequest("PUT", requestURL, nil)
	if err != nil {
		return iodine.New(err, nil)
	}
	// by default while creating a bucket make it default "private"
	req.Header.Add("x-amz-acl", "private")

	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return iodine.New(err, nil)
	}
	if res != nil {
		if res.StatusCode != http.StatusOK {
			return iodine.New(ResponseToError(res), nil)
		}
	}
	defer res.Body.Close()
	return nil
}

func (c *s3Client) PutBucketACL(acl string) error {
	if !isValidBucketACL(acl) {
		return iodine.New(InvalidACL{ACL: acl}, nil)
	}
	_, object := c.url2BucketAndObject()
	if object != "" {
		return iodine.New(InvalidQueryURL{URL: ""}, nil)
	}
	requestURL, err := c.getRequestURL()
	if err != nil {
		return iodine.New(err, nil)
	}

	// new request
	u := fmt.Sprintf("%s?acl", requestURL)
	req, err := c.newRequest("PUT", u, nil)
	if err != nil {
		return iodine.New(err, nil)
	}

	// by default without acl while creating a bucket
	// make it default "private"
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
			return iodine.New(ResponseToError(res), nil)
		}
	}
	defer res.Body.Close()
	return nil
}

// Stat - send a 'HEAD' on a bucket or object to get its metadata
func (c *s3Client) Stat() (*client.Content, error) {
	bucket, object := c.url2BucketAndObject()
	return c.getMetadata(bucket, object)
}
