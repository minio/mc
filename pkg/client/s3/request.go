package s3

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio/pkg/iodine"
)

func (c *s3Client) isValidQueryURL(queryURL string) bool {
	u, err := url.Parse(queryURL)
	if err != nil {
		return false
	}
	if !strings.Contains(u.Scheme, "http") {
		return false
	}
	return true
}

// url2BucketAndObject gives bucketName and objectName from URL path
func (c *s3Client) url2BucketAndObject() (bucketName, objectName string) {
	splits := strings.SplitN(c.Path, "/", 3)
	switch len(splits) {
	case 0, 1:
		bucketName = ""
		objectName = ""
	case 2:
		bucketName = splits[1]
		objectName = ""
	case 3:
		bucketName = splits[1]
		objectName = splits[2]
	}
	return bucketName, objectName
}

func (c *s3Client) getBucketRequestURL(bucket string) string {
	return fmt.Sprintf("%s://%s/%s", c.Scheme, c.Host, bucket)
}

// getObjectRequestURL constructs a URL using bucket and object
func (c *s3Client) getObjectRequestURL(bucket, object string) string {
	return c.getBucketRequestURL(bucket) + "/" + object
}

func (c *s3Client) mustGetRequestURL() string {
	requestURL, _ := c.getRequestURL()
	return requestURL
}

// getRequestURL constructs a URL. URL is appropriately encoded based on the host's object storage implementation.
func (c *s3Client) getRequestURL() (string, error) {
	bucket, object := c.url2BucketAndObject()
	// Avoid bucket names with "." in them
	// Citing - http://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html
	//
	// =====================
	// When using virtual hosted–style buckets with SSL, the SSL wild card certificate
	// only matches buckets that do not contain periods. To work around this, use HTTP
	// or write your own certificate verification logic.
	// =====================
	//
	switch {
	case bucket == "" && object == "":
		return fmt.Sprintf("%s://%s/", c.Scheme, c.Host), nil
	case bucket != "" && object == "":
		return c.getBucketRequestURL(bucket), nil
	case bucket != "" && object != "":
		return c.getObjectRequestURL(bucket, object), nil
	}
	return "", iodine.New(errors.New("Unexpected error, please report this.."), nil)
}

// Instantiate a new request
func (c *s3Client) newRequest(method, url string, body io.ReadCloser) (*http.Request, error) {
	errParams := map[string]string{
		"url":       url,
		"method":    method,
		"userAgent": c.UserAgent,
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, iodine.New(err, errParams)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format(http.TimeFormat))
	return req, nil
}
