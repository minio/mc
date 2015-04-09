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
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"net/url"
	"regexp"
	"strings"

	"encoding/xml"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

type listBucketResults struct {
	Contents       []*client.Item
	IsTruncated    bool
	MaxKeys        int
	Name           string // bucket name
	Marker         string
	Delimiter      string
	Prefix         string
	CommonPrefixes []*client.Prefix
}

// Meta holds Amazon S3 client credentials and flags.
type Meta struct {
	*Auth
	Transport http.RoundTripper // or nil for the default behavior
}

// Auth - see http://docs.amazonwebservices.com/AmazonS3/latest/dev/index.html?RESTAuthentication.html
type Auth struct {
	AccessKeyID     string
	SecretAccessKey string
	// Used for SSL transport layer
	CertPEM string
	KeyPEM  string
}

// TLSConfig - TLS cert and key configuration
type TLSConfig struct {
	CertPEMBlock []byte
	KeyPEMBlock  []byte
}

type s3Client struct {
	*Meta

	// Supports URL in following formats
	//  - http://<ipaddress>/<bucketname>/<object>
	//  - http://<bucketname>.<domain>/<object>
	*url.URL
}

// GetNewClient returns an initialized s3Client structure.
func GetNewClient(auth *Auth, urlStr string, transport http.RoundTripper) client.Client {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}
	s3c := &s3Client{
		&Meta{
			Auth:      auth,
			Transport: GetNewTraceTransport(s3Verify{}, transport),
		}, u,
	}
	return s3c
}

// bucketURL constructs a URL (with a trailing slash) for a given
// bucket. URL is appropriately encoded based on the host's object
// storage implementation.
func (c *s3Client) bucketURL(bucket string) string {
	var url string
	// TODO: Bucket names can contain ".".  This second check should be removed.
	// when minio server supports buckets with "."
	if IsValidBucketName(bucket) && !strings.Contains(bucket, ".") {
		// if localhost use PathStyle
		if strings.Contains(c.Host, "localhost") || strings.Contains(c.Host, "127.0.0.1") {
			return fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
		}
		// Verify if its ip address, use PathStyle
		host, _, _ := net.SplitHostPort(c.Host)
		if net.ParseIP(host) != nil {
			return fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
		}
		// For DNS hostname or amazonaws.com use subdomain style
		url = fmt.Sprintf("%s://%s.%s/", c.Scheme, bucket, c.Host)
	}
	return url
}

// keyURL constructs a URL using bucket and object key
func (c *s3Client) keyURL(bucket, key string) string {
	url := c.bucketURL(bucket)
	if strings.Contains(c.Host, "localhost") || strings.Contains(c.Host, "127.0.0.1") {
		return url + "/" + key
	}
	host, _, _ := net.SplitHostPort(c.Host)
	if net.ParseIP(host) != nil {
		return url + "/" + key
	}
	return url + key
}

func newReq(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// TODO: never exit from inside a package. Let the
		// caller handle errors gracefully.
		panic(fmt.Sprintf("s3 client; invalid URL: %v", err))
	}
	req.Header.Set("User-Agent", "Minio s3Client")
	return req
}

func listAllMyBuckets(r io.Reader) ([]*client.Bucket, error) {
	type allMyBuckets struct {
		Buckets struct {
			Bucket []*client.Bucket
		}
	}
	var res allMyBuckets
	if err := xml.NewDecoder(r).Decode(&res); err != nil {
		return nil, iodine.New(err, nil)
	}
	return res.Buckets.Bucket, nil
}

// IsValidBucketName reports whether bucket is a valid bucket name, per Amazon's naming restrictions.
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
func IsValidBucketName(bucket string) bool {
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
