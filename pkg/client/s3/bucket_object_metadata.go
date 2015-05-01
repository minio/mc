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
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

func (c *s3Client) getMetadata(bucket, object string) (content *client.Content, err error) {
	if object == "" {
		return c.getBucketMetadata(bucket)
	}
	return c.getObjectMetadata(bucket, object)
}

func (c *s3Client) getBucketMetadata(bucket string) (content *client.Content, err error) {
	queryURL, err := c.getRequestURL()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if !c.isValidQueryURL(queryURL) {
		return nil, iodine.New(InvalidQueryURL{URL: queryURL}, nil)
	}
	req, err := c.newRequest("HEAD", queryURL, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	content = new(client.Content)
	content.Name = bucket
	content.FileType = os.ModeDir

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		fallthrough
	case http.StatusMovedPermanently:
		return content, nil
	default:
		return nil, iodine.New(NewError(res), nil)
	}
}

// getObjectMetadata - returns nil, os.ErrNotExist if not on object storage
func (c *s3Client) getObjectMetadata(bucket, object string) (content *client.Content, err error) {
	queryURL, err := c.getRequestURL()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if !c.isValidQueryURL(queryURL) {
		return nil, iodine.New(InvalidQueryURL{URL: queryURL}, nil)
	}
	req, err := c.newRequest("HEAD", queryURL, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusNotFound:
		return nil, iodine.New(ObjectNotFound{Bucket: bucket, Object: object}, nil)
	case http.StatusOK:
		// verify for Content-Length
		contentLength, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		// AWS S3 uses RFC1123 standard for Date in HTTP header
		date, err := time.Parse(time.RFC1123, res.Header.Get("Last-Modified"))
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		content = new(client.Content)
		content.Name = object
		content.Time = date
		content.Size = contentLength
		content.FileType = 0
		return content, nil
	default:
		return nil, iodine.New(NewError(res), nil)
	}
}
