/*
1;3803;0c * Minio Client (C) 2015 Minio, Inc.
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
	"os"
	"strings"
	"time"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

func (c *s3Client) listInGoRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	var contents []*client.Content
	bucket, objectPrefix := c.url2BucketAndObject()
	content, err := c.getObjectMetadata(bucket, objectPrefix)
	switch err {
	case nil: // List a single object. Exact key
		contentCh <- client.ContentOnChannel{
			Content: content,
			Err:     nil,
		}
	default:
		if bucket == "" {
			contents, err = c.listBuckets()
			if err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     iodine.New(err, nil),
				}
				return
			}
			for _, content := range contents {
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
			return
		}
		// List all objects matching the key prefix
		contents, err = c.listObjects(bucket, "", objectPrefix, "/", globalMaxKeys)
		if err != nil {
			contentCh <- client.ContentOnChannel{
				Content: nil,
				Err:     iodine.New(err, nil),
			}
			return
		}
		for _, content := range contents {
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	}
}

func (c *s3Client) listRecursiveInGoRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)

	var contents []*client.Content
	bucket, objectPrefix := c.url2BucketAndObject()
	content, err := c.getObjectMetadata(bucket, objectPrefix)
	switch err {
	case nil: // List a single object. Exact key
		contentCh <- client.ContentOnChannel{
			Content: content,
			Err:     nil,
		}
	default:
		if bucket == "" {
			contents, err = c.listBuckets()
			if err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     iodine.New(err, nil),
				}
				return
			}
			for _, content := range contents {
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
			return
		}
		// List all objects matching the key prefix
		contents, err = c.listObjects(bucket, "", objectPrefix, "", globalMaxKeys)
		if err != nil {
			contentCh <- client.ContentOnChannel{
				Content: nil,
				Err:     iodine.New(err, nil),
			}
			return
		}
		for _, content := range contents {
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	}
}

// populate s3 response and decode results into listBucketResults{}
func (c *s3Client) decodeBucketResults(queryURL string) (*listBucketResults, error) {
	if !c.isValidQueryURL(queryURL) {
		return nil, iodine.New(InvalidQueryURL{URL: queryURL}, nil)
	}
	bres := &listBucketResults{}
	req, err := c.newRequest("GET", queryURL, nil)
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
	if res.StatusCode != http.StatusOK {
		return nil, iodine.New(NewError(res), nil)
	}
	var logbuf bytes.Buffer
	err = xml.NewDecoder(io.TeeReader(res.Body, &logbuf)).Decode(bres)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"XMLError": logbuf.String()})
	}
	return bres, nil
}

// filter contents out of content and provide marker for future request
func (c *s3Client) filterContents(startAt, marker, prefix, delimiter string, cts []*content) ([]*client.Content, string, error) {
	var contents []*client.Content
	var nextMarker string
	for _, ct := range cts {
		if ct.Key == marker && ct.Key != startAt {
			// Skip first dup on pages 2 and higher.
			continue
		}
		if ct.Key < startAt {
			msg := fmt.Sprintf("Unexpected response from Amazon: content key %q but wanted greater than %q", ct.Key, startAt)
			return nil, marker, iodine.New(client.UnexpectedError{Err: errors.New(msg)}, nil)
		}
		content := new(client.Content)
		content.Name = ct.Key
		content.Time = ct.LastModified
		content.Size = ct.Size
		content.FileType = 0
		contents = append(contents, content)
		nextMarker = ct.Key
	}
	return contents, nextMarker, nil
}

// Populare query URL for Listobjects requests
func (c *s3Client) getQueryURL(marker, prefix, delimiter string, fetchN int) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s?max-keys=%d", c.mustGetRequestURL(), fetchN))
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
	return buffer.String()
}

// listObjects returns 0 to maxKeys (inclusive) contents from the
// provided bucket. Keys before startAt will be skipped. (This is the S3
// 'marker' value). If the length of the returned contents is equal to
// maxKeys, there is no indication whether or not the returned list is truncated.
func (c *s3Client) listObjects(bucket, startAt, prefix, delimiter string, maxKeys int) (contents []*client.Content, err error) {
	if maxKeys <= 0 {
		return nil, iodine.New(InvalidMaxKeys{MaxKeys: maxKeys}, nil)
	}
	marker := startAt
	for len(contents) < maxKeys {
		fetchN := maxKeys - len(contents)
		if fetchN > globalMaxKeys {
			fetchN = globalMaxKeys
		}
		bres, err := c.decodeBucketResults(c.getQueryURL(marker, prefix, delimiter, fetchN))
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		if bres.MaxKeys != fetchN || bres.Name != bucket || bres.Marker != marker {
			msg := fmt.Sprintf("Unexpected parse from server: %#v", bres)
			return nil, iodine.New(client.UnexpectedError{
				Err: errors.New(msg)}, nil)
		}
		contents, marker, err = c.filterContents(startAt, marker, prefix, delimiter, bres.Contents)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		for _, prefix := range bres.CommonPrefixes {
			content := &client.Content{
				Name: prefix.Prefix,
				// TODO no way of fixiing this as of now
				Time:     time.Now(),
				Size:     0,
				FileType: os.ModeDir,
			}
			contents = append(contents, content)
		}
		if !bres.IsTruncated {
			break
		}
		if len(contents) == 0 {
			errMsg := errors.New("No contents replied")
			return nil, iodine.New(client.UnexpectedError{Err: errMsg}, nil)
		}
	}
	return contents, nil
}

// Get list of buckets
func (c *s3Client) listBuckets() ([]*client.Content, error) {
	requestURL, err := c.getRequestURL()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	req, err := c.newRequest("GET", requestURL, nil)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	// do not ignore signatures for 'listBuckets()' it is never a public request for amazon s3
	// so lets aggressively verify
	if strings.Contains(c.Host, "amazonaws.com") && (c.AccessKeyID == "" || c.SecretAccessKey == "") {
		msg := "Authorization key cannot be empty for listing buckets, please choose a valid bucketname if its a public request"
		return nil, iodine.New(errors.New(msg), nil)
	}
	// rest we can ignore
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		c.signRequest(req, c.Host)
	}
	res, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if res != nil {
		if res.StatusCode != http.StatusOK {
			err = NewError(res)
			return nil, iodine.New(err, nil)
		}
	}
	defer res.Body.Close()

	type bucket struct {
		Name         string
		CreationDate time.Time
	}
	type allMyBuckets struct {
		Buckets struct {
			Bucket []*bucket
		}
	}
	var buckets allMyBuckets
	if err := xml.NewDecoder(res.Body).Decode(&buckets); err != nil {
		return nil, iodine.New(client.UnexpectedError{
			Err: errors.New("Malformed response received from server")},
			map[string]string{"XMLError": err.Error()})
	}
	var contents []*client.Content
	for _, b := range buckets.Buckets.Bucket {
		content := new(client.Content)
		content.Name = b.Name
		content.Time = b.CreationDate
		content.FileType = os.ModeDir
		contents = append(contents, content)
	}
	return contents, nil
}
