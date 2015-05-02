package s3

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

func (c *s3Client) getMetadata(bucket, object string) (item *client.Item, err error) {
	if object == "" {
		return c.getBucketMetadata(bucket)
	}
	return c.getObjectMetadata(bucket, object)
}

func (c *s3Client) getBucketMetadata(bucket string) (item *client.Item, err error) {
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
	item = new(client.Item)
	item.Name = bucket
	item.FileType = os.ModeDir

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		fallthrough
	case http.StatusMovedPermanently:
		return item, nil
	default:
		return nil, iodine.New(NewError(res), nil)
	}
}

// getObjectMetadata - returns nil, os.ErrNotExist if not on object storage
func (c *s3Client) getObjectMetadata(bucket, object string) (item *client.Item, err error) {
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
		item = new(client.Item)
		item.Name = object
		item.Time = date
		item.Size = contentLength
		item.FileType = 0
		return item, nil
	default:
		return nil, iodine.New(NewError(res), nil)
	}
}
