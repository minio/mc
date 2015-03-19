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

package s3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"encoding/xml"
	"net/http"
	"net/url"
)

// bySize implements sort.Interface for []Item based on the Size field.
type bySize []*Item

func (a bySize) Len() int           { return len(a) }
func (a bySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySize) Less(i, j int) bool { return a[i].Size < a[j].Size }

/// Bucket API operations

// ListBuckets - Get list of buckets
func (c *Client) ListBuckets() ([]*Bucket, error) {
	url := fmt.Sprintf("%s://%s/", c.Scheme, c.Host)
	req := newReq(url)
	c.Auth.signRequest(req, c.Host)

	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, NewError(res)
	}

	return parseListAllMyBuckets(res.Body)
}

// PutBucket - create new bucket
func (c *Client) PutBucket(bucket string) error {
	var url string
	if IsValidBucket(bucket) && !strings.Contains(bucket, ".") {
		url = fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	}
	req := newReq(url)
	req.Method = "PUT"
	c.Auth.signRequest(req, c.Host)
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return NewError(res)
	}

	return nil
}

// ListObjects returns 0 to maxKeys (inclusive) items from the
// provided bucket. Keys before startAt will be skipped. (This is the S3
// 'marker' value). If the length of the returned items is equal to
// maxKeys, there is no indication whether or not the returned list is truncated.
func (c *Client) ListObjects(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*Item, prefixes []*Prefix, err error) {
	var urlReq string
	var buffer bytes.Buffer

	if maxKeys <= 0 {
		return nil, nil, errors.New("negative maxKeys are invalid")
	}

	marker := startAt
	for len(items) < maxKeys {
		fetchN := maxKeys - len(items)
		if fetchN > MaxKeys {
			fetchN = MaxKeys
		}
		var bres listBucketResults
		buffer.WriteString(fmt.Sprintf("%s?max-keys=%d", c.bucketURL(bucket), fetchN))
		switch true {
		case marker != "":
			buffer.WriteString(fmt.Sprintf("&marker=%s", url.QueryEscape(marker)))
			fallthrough
		case prefix != "":
			buffer.WriteString(fmt.Sprintf("&prefix=%s", url.QueryEscape(prefix)))
			fallthrough
		case delimiter != "":
			buffer.WriteString(fmt.Sprintf("&delimiter=%s", url.QueryEscape(delimiter)))
		}

		urlReq = buffer.String()
		// Try the enumerate three times, since Amazon likes to close
		// https connections a lot, and Go sucks at dealing with it:
		// https://code.google.com/p/go/issues/detail?id=3514
		const maxTries = 5
		for try := 1; try <= maxTries; try++ {
			time.Sleep(time.Duration(try-1) * 100 * time.Millisecond)
			req := newReq(urlReq)
			c.Auth.signRequest(req, c.Host)
			res, err := c.Transport.RoundTrip(req)
			if err != nil {
				if try < maxTries {
					continue
				}
				return nil, nil, err
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				return nil, nil, NewError(res)
			}

			bres = listBucketResults{}
			var logbuf bytes.Buffer
			err = xml.NewDecoder(io.TeeReader(res.Body, &logbuf)).Decode(&bres)
			if err != nil {
				fmt.Printf("Error parsing s3 XML response: %v for %q", err, logbuf.Bytes())
			} else if bres.MaxKeys != fetchN || bres.Name != bucket || bres.Marker != marker {
				msg := fmt.Sprintf("Unexpected parse from server: %#v from: %s", bres, logbuf.Bytes())
				err = errors.New(msg)
				fmt.Print(err)
			}
			if err != nil {
				if try < maxTries-1 {
					continue
				}
				fmt.Print(err)
				return nil, nil, err
			}
			break
		}
		for _, it := range bres.Contents {
			if it.Key == marker && it.Key != startAt {
				// Skip first dup on pages 2 and higher.
				continue
			}
			if it.Key < startAt {
				msg := fmt.Sprintf("Unexpected response from Amazon: item key %q but wanted greater than %q", it.Key, startAt)
				return nil, nil, errors.New(msg)
			}
			items = append(items, it)
			marker = it.Key
		}

		for _, pre := range bres.CommonPrefixes {
			if pre.Prefix != "" {
				prefixes = append(prefixes, pre)
			}
		}

		if !bres.IsTruncated {
			break
		}

		if len(items) == 0 {
			return nil, nil, errors.New("No items replied")
		}
	}
	sort.Sort(bySize(items))
	return items, prefixes, nil
}
