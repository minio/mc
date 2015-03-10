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
 * Mini Object Storage, (C) 2015 Minio, Inc.
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
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"crypto/md5"
	"encoding/base64"
	"io/ioutil"
	"net/http"
)

/// Object API operations

// Put - upload new object to bucket
func (c *Client) Put(bucket, key string, size int64, contents io.Reader) error {
	req := newReq(c.keyURL(bucket, key))
	req.Method = "PUT"
	req.ContentLength = size

	h := md5.New()
	// Memory where data is present
	sink := new(bytes.Buffer)
	mw := io.MultiWriter(h, sink)
	written, err := io.Copy(mw, contents)
	if written != size {
		return fmt.Errorf("Data read mismatch")
	}
	if err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(sink)
	b64 := base64.StdEncoding.EncodeToString(h.Sum(nil))
	req.Header.Set("Content-MD5", b64)
	c.Auth.signRequest(req)

	res, err := c.transport().RoundTrip(req)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		// res.Write(os.Stderr)
		return fmt.Errorf("Got response code %d from s3", res.StatusCode)
	}
	return nil
}

// Stat - returns 0, "", os.ErrNotExist if not on S3
func (c *Client) Stat(key, bucket string) (size int64, date time.Time, reterr error) {
	req := newReq(c.keyURL(bucket, key))
	req.Method = "HEAD"
	c.Auth.signRequest(req)
	res, err := c.transport().RoundTrip(req)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return 0, date, err
	}

	switch res.StatusCode {
	case http.StatusNotFound:
		return 0, date, os.ErrNotExist
	case http.StatusOK:
		size, err = strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return 0, date, err
		}
		if dateStr := res.Header.Get("Last-Modified"); dateStr != "" {
			// AWS S3 uses RFC1123 standard for Date in HTTP header, unlike XML content
			date, err := time.Parse(time.RFC1123, dateStr)
			if err != nil {
				return 0, date, err
			}
			return size, date, nil
		}
	default:
		return 0, date, fmt.Errorf("s3: Unexpected status code %d statting object %v", res.StatusCode, key)
	}
	return
}

// Get - download a requested object from a given bucket
func (c *Client) Get(bucket, key string) (body io.ReadCloser, size int64, err error) {
	req := newReq(c.keyURL(bucket, key))
	c.Auth.signRequest(req)
	res, err := c.transport().RoundTrip(req)
	if err != nil {
		return
	}
	switch res.StatusCode {
	case http.StatusOK:
		return res.Body, res.ContentLength, nil
	case http.StatusNotFound:
		res.Body.Close()
		return nil, 0, os.ErrNotExist
	default:
		res.Body.Close()
		return nil, 0, fmt.Errorf("Amazon HTTP error on GET: %d", res.StatusCode)
	}
}

// GetPartial fetches part of the s3 key object in bucket.
// If length is negative, the rest of the object is returned.
// The caller must close rc.
func (c *Client) GetPartial(bucket, key string, offset, length int64) (rc io.ReadCloser, err error) {
	if offset < 0 {
		return nil, fmt.Errorf("invalid negative length")
	}

	req := newReq(c.keyURL(bucket, key))
	if length >= 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	c.Auth.signRequest(req)

	res, err := c.transport().RoundTrip(req)
	if err != nil {
		return
	}
	switch res.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		return res.Body, nil
	case http.StatusNotFound:
		res.Body.Close()
		return nil, os.ErrNotExist
	default:
		res.Body.Close()
		return nil, fmt.Errorf("Amazon HTTP error on GET: %d", res.StatusCode)
	}
}
