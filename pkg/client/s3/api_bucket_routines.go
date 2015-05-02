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

func (c *s3Client) listInGoRoutine(itemCh chan client.ItemOnChannel) {
	defer close(itemCh)
	var items []*client.Item
	bucket, objectPrefix := c.url2BucketAndObject()
	item, err := c.getObjectMetadata(bucket, objectPrefix)
	switch err {
	case nil: // List a single object. Exact key
		itemCh <- client.ItemOnChannel{
			Item: item,
			Err:  nil,
		}
	default:
		if bucket == "" {
			items, err = c.listBuckets()
			if err != nil {
				itemCh <- client.ItemOnChannel{
					Item: nil,
					Err:  iodine.New(err, nil),
				}
				return
			}
			for _, item := range items {
				itemCh <- client.ItemOnChannel{
					Item: item,
					Err:  nil,
				}
			}
			return
		}
		// List all objects matching the key prefix
		items, err = c.listObjects(bucket, "", objectPrefix, "/", globalMaxKeys)
		if err != nil {
			itemCh <- client.ItemOnChannel{
				Item: nil,
				Err:  iodine.New(err, nil),
			}
			return
		}
		for _, item := range items {
			itemCh <- client.ItemOnChannel{
				Item: item,
				Err:  nil,
			}
		}
	}
}

func (c *s3Client) listRecursiveInGoRoutine(itemCh chan client.ItemOnChannel) {
	defer close(itemCh)

	var items []*client.Item
	bucket, objectPrefix := c.url2BucketAndObject()
	item, err := c.getObjectMetadata(bucket, objectPrefix)
	switch err {
	case nil: // List a single object. Exact key
		itemCh <- client.ItemOnChannel{
			Item: item,
			Err:  nil,
		}
	default:
		if bucket == "" {
			items, err = c.listBuckets()
			if err != nil {
				itemCh <- client.ItemOnChannel{
					Item: nil,
					Err:  iodine.New(err, nil),
				}
				return
			}
			for _, item := range items {
				itemCh <- client.ItemOnChannel{
					Item: item,
					Err:  nil,
				}
			}
			return
		}
		// List all objects matching the key prefix
		items, err = c.listObjects(bucket, "", objectPrefix, "", globalMaxKeys)
		if err != nil {
			itemCh <- client.ItemOnChannel{
				Item: nil,
				Err:  iodine.New(err, nil),
			}
			return
		}
		for _, item := range items {
			itemCh <- client.ItemOnChannel{
				Item: item,
				Err:  nil,
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

// filter items out of content and provide marker for future request
func (c *s3Client) filterItems(startAt, marker, prefix, delimiter string, contents []*content) (items []*client.Item, nextMarker string, err error) {
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
		item.Name = it.Key
		item.Time = it.LastModified
		item.Size = it.Size
		item.FileType = 0
		items = append(items, item)
		nextMarker = it.Key
	}
	return items, nextMarker, nil
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

// listObjects returns 0 to maxKeys (inclusive) items from the
// provided bucket. Keys before startAt will be skipped. (This is the S3
// 'marker' value). If the length of the returned items is equal to
// maxKeys, there is no indication whether or not the returned list is truncated.
func (c *s3Client) listObjects(bucket, startAt, prefix, delimiter string, maxKeys int) (items []*client.Item, err error) {
	if maxKeys <= 0 {
		return nil, iodine.New(InvalidMaxKeys{MaxKeys: maxKeys}, nil)
	}
	marker := startAt
	for len(items) < maxKeys {
		fetchN := maxKeys - len(items)
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
		items, marker, err = c.filterItems(startAt, marker, prefix, delimiter, bres.Contents)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		for _, prefix := range bres.CommonPrefixes {
			item := &client.Item{
				Name: prefix.Prefix,
				// TODO no way of fixiing this as of now
				Time:     time.Now(),
				Size:     0,
				FileType: os.ModeDir,
			}
			items = append(items, item)
		}
		if !bres.IsTruncated {
			break
		}
		if len(items) == 0 {
			errMsg := errors.New("No items replied")
			return nil, iodine.New(client.UnexpectedError{Err: errMsg}, nil)
		}
	}
	return items, nil
}

// Get list of buckets
func (c *s3Client) listBuckets() ([]*client.Item, error) {
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
	var items []*client.Item
	for _, b := range buckets.Buckets.Bucket {
		item := new(client.Item)
		item.Name = b.Name
		item.Time = b.CreationDate
		item.FileType = os.ModeDir
		items = append(items, item)
	}
	return items, nil
}
