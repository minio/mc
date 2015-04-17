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
	"io"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

/// Object API operations

// GetObjectMetadata - returns nil, os.ErrNotExist if not on object storage
func (c *s3Client) GetObjectMetadata(bucket, key string) (item *client.Item, reterr error) {
	if bucket == "" || key == "" {
		return nil, iodine.New(client.InvalidArgument{}, nil)
	}
	req, err := newReq(c.keyURL(bucket, key), c.UserAgent, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	req.Method = "HEAD"
	c.signRequest(req, c.Host)
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusNotFound:
		return nil, iodine.New(client.ObjectNotFound{Bucket: bucket, Object: key}, nil)
	case http.StatusOK:
		contentLength, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		date, err := time.Parse(time.RFC1123, res.Header.Get("Last-Modified"))
		// AWS S3 uses RFC1123 standard for Date in HTTP header, unlike XML content
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		item = new(client.Item)
		item.Key = key
		item.LastModified = date
		item.Size = contentLength
		item.ETag = strings.Trim(res.Header.Get("ETag"), "\"")
		return item, nil
	default:
		return nil, iodine.New(NewError(res), nil)
	}
}

// Get - download a requested object from a given bucket
func (c *s3Client) Get(bucket, key string) (body io.ReadCloser, size int64, md5 string, err error) {
	req, err := newReq(c.keyURL(bucket, key), c.UserAgent, nil)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	c.signRequest(req, c.Host)
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

// GetPartial fetches part of the s3 key object in bucket.
// If length is negative, the rest of the object is returned.
func (c *s3Client) GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	if offset < 0 {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	req, err := newReq(c.keyURL(bucket, key), c.UserAgent, nil)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	if length >= 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	c.signRequest(req, c.Host)

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
