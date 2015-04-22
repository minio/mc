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
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

/// Bucket API operations

// ListObjects - list objects inside a bucket or with prefix
func (c *s3Client) List() (items []*client.Item, err error) {
	bucket, objectPrefix := c.url2Object()
	item, err := c.GetObjectMetadata()
	switch err {
	case nil: // List a single object. Exact key
		items = append(items, item)
		return items, nil
	default:
		// if not bucket provided return list of all buckets
		if bucket == "" {
			return c.listBucketsInternal()
		}
		// List all objects matching the key prefix
		items, err = c.listObjectsInternal(bucket, "", objectPrefix, "", globalMaxKeys)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		// even if items are equal to '0' is valid case
		return items, nil
	}
}

// populate s3 response and decode results into listBucketResults{}
func (c *s3Client) decodeBucketResults(urlReq string) (listBucketResults, error) {
	bres := listBucketResults{}
	req, err := c.getNewReq(urlReq, nil)
	if err != nil {
		return listBucketResults{}, iodine.New(err, nil)
	}
	c.signRequest(req, c.Host)
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return listBucketResults{}, iodine.New(err, nil)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return listBucketResults{}, iodine.New(NewError(res), nil)
	}

	var logbuf bytes.Buffer
	err = xml.NewDecoder(io.TeeReader(res.Body, &logbuf)).Decode(&bres)
	if err != nil {
		return listBucketResults{}, iodine.New(err, map[string]string{"XMLError": logbuf.String()})
	}
	return bres, nil
}

// filter items out of content and provide marker for future request
func (c *s3Client) filterItems(startAt, marker, prefix, delimiter string, contents []*item) (items []*client.Item, nextMarker string, err error) {
	for _, it := range contents {
		if it.Key == marker && it.Key != startAt {
			// Skip first dup on pages 2 and higher.
			continue
		}
		if it.Key < startAt {
			msg := fmt.Sprintf("Unexpected response from Amazon: item key %q but wanted greater than %q", it.Key, startAt)
			return nil, marker, iodine.New(client.UnexpectedError{Err: errors.New(msg)}, nil)
		}
		item := new(client.Item)
		// TODO (y4m4) - this is temporary, fix this after passing down proper delimiters
		// for now using filepath.Separator
		item.Name = strings.TrimPrefix(it.Key, filepath.Clean(prefix)+string(filepath.Separator))
		item.Time = it.LastModified
		item.Size = it.Size
		items = append(items, item)
		nextMarker = it.Key
	}
	return items, nextMarker, nil
}

// listObjectsInternal returns 0 to maxKeys (inclusive) items from the
// provided bucket. Keys before startAt will be skipped. (This is the S3
// 'marker' value). If the length of the returned items is equal to
// maxKeys, there is no indication whether or not the returned list is truncated.
func (c *s3Client) listObjectsInternal(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*client.Item, err error) {
	var urlReq string
	var buffer bytes.Buffer
	if maxKeys <= 0 {
		return nil, iodine.New(InvalidMaxKeys{MaxKeys: maxKeys}, nil)
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
		bres, err = c.decodeBucketResults(urlReq)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		if bres.MaxKeys != fetchN || bres.Name != bucket || bres.Marker != marker {
			msg := fmt.Sprintf("Unexpected parse from server: %#v", bres)
			return nil, iodine.New(client.UnexpectedError{
				Err: errors.New(msg)}, nil)
		}
		items, marker, err = c.filterItems(startAt, marker, prefix, delimiter, bres.Contents)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		if !bres.IsTruncated {
			break
		}

		if len(items) == 0 {
			errMsg := errors.New("No items replied")
			return nil, iodine.New(client.UnexpectedError{
				Err: errMsg}, nil)
		}
	}
	sort.Sort(client.BySize(items))
	return items, nil
}
