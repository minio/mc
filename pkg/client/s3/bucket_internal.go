package s3

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// Try the enumerate 5 times, since Amazon likes to close
// https connections a lot, and Go sucks at dealing with it:
// https://code.google.com/p/go/issues/detail?id=3514
func (c *s3Client) retryRequest(urlReq string) (listBucketResults, error) {
	const maxTries = 5
	bres := listBucketResults{}
	for try := 1; try <= maxTries; try++ {
		time.Sleep(time.Duration(try-1) * 100 * time.Millisecond)
		req := newReq(urlReq, c.UserAgent)
		c.signRequest(req, c.Host)
		res, err := c.Transport.RoundTrip(req)
		if err != nil {
			if try < maxTries {
				continue
			}
			return listBucketResults{}, iodine.New(err, nil)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return listBucketResults{}, iodine.New(NewError(res), nil)
		}

		var logbuf bytes.Buffer
		err = xml.NewDecoder(io.TeeReader(res.Body, &logbuf)).Decode(&bres)
		if err != nil {
			fmt.Printf("Error parsing s3 XML response: %v for %q\n", err, logbuf.Bytes())
			if try < maxTries-1 {
				fmt.Printf("Reconnecting...\n")
				continue
			}
			return listBucketResults{}, iodine.New(err, nil)
		}
		break
	}
	return bres, nil
}

// get items out of content and provide marker for future request
func (c *s3Client) getItems(s, m string, contents []*client.Item) (items []*client.Item, marker string, err error) {
	for _, it := range contents {
		if it.Key == m && it.Key != s {
			// Skip first dup on pages 2 and higher.
			continue
		}
		if it.Key < s {
			msg := fmt.Sprintf("Unexpected response from Amazon: item key %q but wanted greater than %q", it.Key, s)
			return nil, m, iodine.New(errors.New(msg), nil)
		}
		items = append(items, it)
		marker = it.Key
	}
	return items, marker, nil
}

// queryObjects returns 0 to maxKeys (inclusive) items from the
// provided bucket. Keys before startAt will be skipped. (This is the S3
// 'marker' value). If the length of the returned items is equal to
// maxKeys, there is no indication whether or not the returned list is truncated.
func (c *s3Client) queryObjects(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*client.Item, prefixes []*client.Prefix, err error) {
	var urlReq string
	var buffer bytes.Buffer

	if maxKeys <= 0 {
		return nil, nil, iodine.New(client.InvalidMaxKeys{MaxKeys: maxKeys}, nil)
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
		bres, err = c.retryRequest(urlReq)
		if err != nil {
			return nil, nil, iodine.New(err, nil)
		}
		if bres.MaxKeys != fetchN || bres.Name != bucket || bres.Marker != marker {
			msg := fmt.Sprintf("Unexpected parse from server: %#v", bres)
			err = errors.New(msg)
			return nil, nil, iodine.New(err, nil)
		}
		items, marker, err = c.getItems(startAt, marker, bres.Contents)
		if err != nil {
			return nil, nil, iodine.New(err, nil)
		}
		prefixes = (bres.CommonPrefixes)

		if !bres.IsTruncated {
			break
		}

		if len(items) == 0 {
			return nil, nil, iodine.New(errors.New("No items replied"), nil)
		}
	}
	sort.Sort(bySize(items))
	return items, prefixes, nil
}
