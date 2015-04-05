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
	"os"
	"sort"
	"strings"
	"time"

	"encoding/xml"
	"net/http"
	"net/url"

	"github.com/minio-io/mc/pkg/client"
)

// bySize implements sort.Interface for []Item based on the Size field.
type bySize []*client.Item

func (a bySize) Len() int           { return len(a) }
func (a bySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySize) Less(i, j int) bool { return a[i].Size < a[j].Size }

/// Bucket API operations

// ListBuckets - Get list of buckets
func (c *s3Client) ListBuckets() ([]*client.Bucket, error) {
	url := fmt.Sprintf("%s://%s/", c.Scheme, c.Host)
	req := newReq(url)
	c.signRequest(req, c.Host)

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
func (c *s3Client) PutBucket(bucket string) error {
	var url string
	if IsValidBucketName(bucket) && !strings.Contains(bucket, ".") {
		url = fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
	}
	req := newReq(url)
	req.Method = "PUT"
	c.signRequest(req, c.Host)
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

// Try the enumerate 5 times, since Amazon likes to close
// https connections a lot, and Go sucks at dealing with it:
// https://code.google.com/p/go/issues/detail?id=3514
func (c *s3Client) retry(urlReq string) (listBucketResults, error) {
	const maxTries = 5
	bres := listBucketResults{}
	for try := 1; try <= maxTries; try++ {
		time.Sleep(time.Duration(try-1) * 100 * time.Millisecond)
		req := newReq(urlReq)
		c.signRequest(req, c.Host)
		res, err := c.Transport.RoundTrip(req)
		if err != nil {
			if try < maxTries {
				continue
			}
			return listBucketResults{}, err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return listBucketResults{}, NewError(res)
		}

		var logbuf bytes.Buffer
		err = xml.NewDecoder(io.TeeReader(res.Body, &logbuf)).Decode(&bres)
		if err != nil {
			fmt.Printf("Error parsing s3 XML response: %v for %q\n", err, logbuf.Bytes())
			if try < maxTries-1 {
				fmt.Printf("Reconnecting...\n")
				continue
			}
			return listBucketResults{}, err
		}
		break
	}
	return bres, nil
}

func (c *s3Client) getItems(s, m string, contents []*client.Item) (items []*client.Item, marker string, err error) {
	for _, it := range contents {
		if it.Key == m && it.Key != s {
			// Skip first dup on pages 2 and higher.
			continue
		}
		if it.Key < s {
			msg := fmt.Sprintf("Unexpected response from Amazon: item key %q but wanted greater than %q", it.Key, s)
			return nil, m, errors.New(msg)
		}
		items = append(items, it)
		marker = it.Key
	}
	return items, marker, nil
}

func (c *s3Client) getPrefixes(commonPrefixes []*client.Prefix) (prefixes []*client.Prefix, err error) {
	for _, pre := range commonPrefixes {
		if pre.Prefix != "" {
			prefixes = append(prefixes, pre)
		}
	}
	return prefixes, nil
}

func (c *s3Client) ListObjects(bucket, objectPrefix string) (items []*client.Item, err error) {
	size, date, err := c.Stat(bucket, objectPrefix)
	switch err {
	case nil: // List a single object. Exact key
		items = append(items, &client.Item{Key: objectPrefix, LastModified: date, Size: size})
		return items, nil
	case os.ErrNotExist:
		// List all objects matching the key prefix
		items, _, err = c.queryObjects(bucket, "", objectPrefix, "", globalMaxKeys)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return items, nil
		}
		return nil, os.ErrNotExist
	default: // Error
		return nil, err
	}
}

// queryObjects returns 0 to maxKeys (inclusive) items from the
// provided bucket. Keys before startAt will be skipped. (This is the S3
// 'marker' value). If the length of the returned items is equal to
// maxKeys, there is no indication whether or not the returned list is truncated.
func (c *s3Client) queryObjects(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*client.Item, prefixes []*client.Prefix, err error) {
	var urlReq string
	var buffer bytes.Buffer

	if maxKeys <= 0 {
		return nil, nil, errors.New("negative maxKeys are invalid")
	}

	marker := startAt
	for len(items) < maxKeys {
		fetchN := maxKeys - len(items)
		if fetchN > globalMaxKeys {
			fetchN = globalMaxKeys
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
		bres, err = c.retry(urlReq)
		if err != nil {
			return nil, nil, err
		}
		if bres.MaxKeys != fetchN || bres.Name != bucket || bres.Marker != marker {
			msg := fmt.Sprintf("Unexpected parse from server: %#v", bres)
			err = errors.New(msg)
			return nil, nil, err
		}
		items, marker, err = c.getItems(startAt, marker, bres.Contents)
		if err != nil {
			return nil, nil, err
		}
		prefixes, err = c.getPrefixes(bres.CommonPrefixes)
		if err != nil {
			return nil, nil, err
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
