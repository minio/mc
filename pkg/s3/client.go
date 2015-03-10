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

// Package s3 implements a generic Amazon S3 client
package s3

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"

	"encoding/xml"
)

// Total max object list
const (
	MaxKeys = 1000
)

// Bucket - carries s3 bucket reply header
type Bucket struct {
	Name         string
	CreationDate xmlTime // 2006-02-03T16:45:09.000Z
}

// Item - object item list
type Item struct {
	Key          string
	LastModified xmlTime
	Size         int64
}

// Prefix - common prefix
type Prefix struct {
	Prefix string
}

type listBucketResults struct {
	Contents       []*Item
	IsTruncated    bool
	MaxKeys        int
	Name           string // bucket name
	Marker         string
	Delimiter      string
	Prefix         string
	CommonPrefixes []*Prefix
}

// Client is an Amazon S3 client.
type Client struct {
	*Auth
	Transport http.RoundTripper // or nil for the default
	Trace     HttpTracer        // if not nil, Tracing will be enabled
}

func NewClient(auth *Auth, transport http.RoundTripper, trace HttpTracer) *Client {
	return &Client{
		Auth:      auth,
		Transport: transport,
		Trace:     trace,
	}
}

func (c *Client) transport() http.RoundTripper {
	if c.Transport != nil {
		return c.Transport
	}
	return http.DefaultTransport
}

// bucketURL returns the URL prefix of the bucket, with trailing slash
func (c *Client) bucketURL(bucket string) string {
	var url string
	if IsValidBucket(bucket) && !strings.Contains(bucket, ".") {
		// if localhost forcePathStyle
		if strings.Contains(c.endpoint(), "localhost") || strings.Contains(c.endpoint(), "127.0.0.1") {
			url = fmt.Sprintf("%s/%s", c.endpoint(), bucket)
			goto ret
		}
		host, _, _ := net.SplitHostPort(c.Endpoint)
		if net.ParseIP(host) != nil {
			url = fmt.Sprintf("%s/%s", c.endpoint(), bucket)
			goto ret
		}
		if !c.S3ForcePathStyle {
			if strings.Contains(c.endpoint(), "amazonaws.com") {
				url = fmt.Sprintf("https://%s.%s/", bucket, strings.TrimPrefix(c.endpoint(), "https://"))
			} else {
				url = fmt.Sprintf("http://%s.%s/", bucket, strings.TrimPrefix(c.endpoint(), "http://"))
			}
		} else {
			url = fmt.Sprintf("%s/%s", c.endpoint(), bucket)
		}
	}

ret:
	return url
}

func (c *Client) keyURL(bucket, key string) string {
	// if localhost forcePathStyle
	host, _, _ := net.SplitHostPort(c.Endpoint)
	ok := (strings.Contains(c.endpoint(), "localhost") || strings.Contains(bucket, "127.0.0.1") || c.S3ForcePathStyle || net.ParseIP(host) != nil)
	if ok {
		return c.bucketURL(bucket) + "/" + key
	}
	return c.bucketURL(bucket) + key
}

func newReq(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(fmt.Sprintf("s3 client; invalid URL: %v", err))
	}
	req.Header.Set("User-Agent", "Minio Client")
	return req
}

func parseListAllMyBuckets(r io.Reader) ([]*Bucket, error) {
	type allMyBuckets struct {
		Buckets struct {
			Bucket []*Bucket
		}
	}
	var res allMyBuckets
	if err := xml.NewDecoder(r).Decode(&res); err != nil {
		return nil, err
	}
	return res.Buckets.Bucket, nil
}

// IsValidBucket reports whether bucket is a valid bucket name, per Amazon's naming restrictions.
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
func IsValidBucket(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if bucket[0] == '.' || bucket[len(bucket)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucket); match == true {
		return false
	}
	// We don't support buckets with '.' in them
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", bucket)
	return match
}
