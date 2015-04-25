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
	"fmt"
	"io"
	"strings"

	"net/http"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

/// Object API operations

// Get - download a requested object from a given bucket
func (c *s3Client) Get() (body io.ReadCloser, size int64, md5 string, err error) {
	bucket, object := c.url2BucketAndObject()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return nil, 0, "", iodine.New(InvalidBucketName{Bucket: bucket}, nil)
	}
	req, err := c.newRequest("GET", c.objectURL(bucket, object), nil)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}

	if res.StatusCode != http.StatusOK {
		return nil, 0, "", iodine.New(NewError(res), nil)
	}
	md5sum := strings.Trim(res.Header.Get("ETag"), "\"") // trim off the erroneous double quotes
	return res.Body, res.ContentLength, md5sum, nil
}

// GetPartial fetches part of the s3 object in bucket.
// If length is negative, the rest of the object is returned.
func (c *s3Client) GetPartial(offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	bucket, object := c.url2BucketAndObject()
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return nil, 0, "", iodine.New(InvalidBucketName{Bucket: bucket}, nil)
	}
	if offset < 0 {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	req, err := c.newRequest("GET", c.objectURL(bucket, object), nil)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	if length >= 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}

	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}

	switch res.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		return res.Body, res.ContentLength, res.Header.Get("ETag"), nil
	default:
		return nil, 0, "", iodine.New(NewError(res), nil)
	}
}
