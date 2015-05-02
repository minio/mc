package s3

import (
	"net/http"
	"net/url"
	"time"

	"github.com/awslabs/aws-sdk-go/service/s3"
)

//
type content struct {
	Key          string
	LastModified time.Time
	ETag         string
	Size         int64
}

// prefix
type prefix struct {
	Prefix string
}

type listBucketResults struct {
	Contents       []*content
	IsTruncated    bool
	MaxKeys        int
	Name           string // bucket name
	Marker         string
	Delimiter      string
	Prefix         string
	CommonPrefixes []*prefix
}

// Meta holds Amazon S3 client credentials and flags.
type Meta struct {
	*Config
	*s3.S3
	Transport http.RoundTripper // or nil for the default behavior
}

// Config - see http://docs.amazonwebservices.com/AmazonS3/latest/dev/index.html?RESTAuthentication.html
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	HostURL         string
	UserAgent       string
	Debug           bool

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
