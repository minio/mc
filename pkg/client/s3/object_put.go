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
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

/// Object Operations PUT - keeping this in a separate file for readability

func (c *s3Client) put(size int64) (*http.Request, error) {
	queryURL, err := c.getRequestURL()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if !c.isValidQueryURL(queryURL) {
		return nil, iodine.New(InvalidQueryURL{URL: queryURL}, nil)
	}
	req, err := c.newRequest("PUT", queryURL, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	req.ContentLength = size
	return req, nil
}

// Put - upload new object to bucket
func (c *s3Client) Put(md5HexString string, size int64) (io.WriteCloser, error) {
	// if size is negative
	if size < 0 {
		return nil, iodine.New(client.InvalidArgument{Err: errors.New("invalid argument")}, nil)
	}
	// starting Pipe session
	r, w := io.Pipe()
	blockingWriter := client.NewBlockingWriteCloser(w)
	go func() {
		req, err := c.put(size)
		if err != nil {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		// set Content-MD5 only if md5 is provided
		if strings.TrimSpace(md5HexString) != "" {
			md5, err := hex.DecodeString(md5HexString)
			if err != nil {
				err := iodine.New(err, nil)
				r.CloseWithError(err)
				blockingWriter.Release(err)
				return
			}
			req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5))
		}
		if c.AccessKeyID != "" && c.SecretAccessKey != "" {
			c.signRequest(req, c.Host)
		}
		req.Body = r
		// this is necessary for debug, since the underlying transport is a wrapper
		res, err := c.Transport.RoundTrip(req)
		if err != nil {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		if res.StatusCode != http.StatusOK {
			err := iodine.New(ResponseToError(res), nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		r.Close()
		blockingWriter.Release(nil)
	}()
	return blockingWriter, nil
}
