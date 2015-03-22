package client

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client - Minio client interface
type Client interface {
	Get(bucket, object string) (body io.ReadCloser, size int64, err error)
	GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, err error)
	Put(bucket, object string, size int64, body io.Reader) error
	Stat(bucket, object string) (size int64, date time.Time, err error)
	PutBucket(bucket string) error
	ListBuckets() ([]*Bucket, error)
	ListObjects(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*Item, prefixes []*Prefix, err error)
}

// Bucket - carries s3 bucket reply header
type Bucket struct {
	Name         string
	CreationDate XMLTime // 2006-02-03T16:45:09.000Z
}

// Item - object item list
type Item struct {
	Key          string
	LastModified XMLTime
	Size         int64
}

// Prefix - common prefix
type Prefix struct {
	Prefix string
}

// Meta holds Amazon S3 client credentials and flags.
type Meta struct {
	*Auth                       // AWS auth credentials
	Transport http.RoundTripper // or nil for the default behavior

	// Supports URL in following formats
	//  - http://<ipaddress>/<bucketname>/<object>
	//  - http://<bucketname>.<domain>/<object>
	URL *url.URL
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
