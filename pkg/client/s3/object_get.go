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
	"io"
	"strings"

	"net/http"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

/// Object API operations

func (c *s3Client) setRange(req *http.Request, offset, length int64) (*http.Request, error) {
	if offset < 0 {
		return nil, iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	if length >= 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	return req, nil
}

func (c *s3Client) get() (*http.Request, error) {
	queryURL, err := c.getRequestURL()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if !c.isValidQueryURL(queryURL) {
		return nil, iodine.New(InvalidQueryURL{URL: queryURL}, nil)
	}
	req, err := c.newRequest("GET", queryURL, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return req, nil
}

// GetObject fetches full object or part of the s3 object in bucket.
// If length is negative, the rest of the object is returned.
func (c *s3Client) GetObject(offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	req, err := c.get()
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	if offset == 0 && length == 0 {
		req, err = c.setRange(req, offset, length)
		if err != nil {
			return nil, 0, "", iodine.New(err, nil)
		}
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
		md5sum := strings.Trim(res.Header.Get("ETag"), "\"") // trim off the erroneous double quotes
		return res.Body, res.ContentLength, md5sum, nil
	default:
		return nil, 0, "", iodine.New(ResponseToError(res), nil)
	}
}
