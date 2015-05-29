/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/minio/mc/pkg/client"
	s3 "github.com/minio/minio-go"
	"github.com/minio/minio/pkg/iodine"
)

// Config - see http://docs.amazonwebservices.com/AmazonS3/latest/dev/index.html?RESTAuthentication.html
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	HostURL         string
	AppName         string
	AppVersion      string
	AppComments     []string
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
	api     s3.API
	hostURL *client.URL
}

// url2Regions s3 region map used by bucket location constraint
var url2Regions = map[string]string{
	"s3-fips-us-gov-west-1.amazonaws.com": "us-gov-west-1",
	"s3.amazonaws.com":                    "us-east-1",
	"s3-us-west-1.amazonaws.com":          "us-west-1",
	"s3-us-west-2.amazonaws.com":          "us-west-2",
	"s3-eu-west-1.amazonaws.com":          "eu-west-1",
	"s3-eu-central-1.amazonaws.com":       "eu-central-1",
	"s3-ap-southeast-1.amazonaws.com":     "ap-southeast-1",
	"s3-ap-southeast-2.amazonaws.com":     "ap-southeast-2",
	"s3-ap-northeast-1.amazonaws.com":     "ap-northeast-1",
	"s3-sa-east-1.amazonaws.com":          "sa-east-1",
	"s3.cn-north-1.amazonaws.com.cn":      "cn-north-1",
}

func getRegion(host string) string {
	return url2Regions[host]
}

// New returns an initialized s3Client structure. if debug use a internal trace transport
func New(config *Config) (client.Client, error) {
	u, err := client.Parse(config.HostURL)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	var transport http.RoundTripper
	switch {
	case config.Debug == true:
		transport = GetNewTraceTransport(NewTrace(), http.DefaultTransport)
	default:
		transport = http.DefaultTransport
	}
	s3Conf := new(s3.Config)
	s3Conf.AccessKeyID = config.AccessKeyID
	s3Conf.SecretAccessKey = config.SecretAccessKey
	s3Conf.Transport = transport
	s3Conf.AddUserAgent(config.AppName, config.AppVersion, config.AppComments...)
	s3Conf.Region = getRegion(u.Host)
	s3Conf.Endpoint = u.Scheme + "://" + u.Host
	api := s3.New(s3Conf)
	return &s3Client{api: api, hostURL: u}, nil
}

// GetObject - get object
func (c *s3Client) GetObject(offset, length uint64) (io.ReadCloser, uint64, error) {
	bucket, object := c.url2BucketAndObject()
	reader, metadata, err := c.api.GetObject(bucket, object, offset, length)
	if err != nil {
		return nil, length, iodine.New(err, nil)
	}
	return reader, uint64(metadata.Size), nil
}

// PutObject - put object
func (c *s3Client) PutObject(size uint64, data io.Reader) error {
	// md5 is purposefully ignored since AmazonS3 does not return proper md5sum
	// for a multipart upload and there is no need to cross verify,
	// invidual parts are properly verified
	bucket, object := c.url2BucketAndObject()
	// TODO - bump individual part size from default, if needed
	// s3.DefaultPartSize = 1024 * 1024 * 100
	err := c.api.PutObject(bucket, object, size, data)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// MakeBucket - make a new bucket
func (c *s3Client) MakeBucket() error {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return iodine.New(InvalidQueryURL{URL: c.hostURL.String()}, nil)
	}
	// location string is intentionally left out
	err := c.api.MakeBucket(bucket, s3.BucketACL("private"), "")
	return iodine.New(err, nil)
}

// SetBucketACL add canned acl's on a bucket
func (c *s3Client) SetBucketACL(acl string) error {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return iodine.New(InvalidQueryURL{URL: c.hostURL.String()}, nil)
	}
	err := c.api.SetBucketACL(bucket, s3.BucketACL(acl))
	return iodine.New(err, nil)
}

// Stat - send a 'HEAD' on a bucket or object to get its metadata
func (c *s3Client) Stat() (*client.Content, error) {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		metadata, err := c.api.StatObject(bucket, object)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		objectMetadata := new(client.Content)
		objectMetadata.Name = c.hostURL.String() // do not change this
		objectMetadata.Time = metadata.LastModified
		objectMetadata.Size = metadata.Size
		objectMetadata.Type = os.FileMode(0664)
		return objectMetadata, nil
	}
	err := c.api.BucketExists(bucket)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	bucketMetadata := new(client.Content)
	bucketMetadata.Name = bucket
	bucketMetadata.Type = os.ModeDir
	return bucketMetadata, nil
}

// url2BucketAndObject gives bucketName and objectName from URL path
func (c *s3Client) url2BucketAndObject() (bucketName, objectName string) {
	splits := strings.SplitN(c.hostURL.Path, "/", 3)
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

/// Bucket API operations

// List - list at delimited path, if not recursive
func (c *s3Client) List(recursive bool) <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	switch recursive {
	case true:
		go c.listRecursiveInRoutine(contentCh)
	default:
		go c.listInRoutine(contentCh)
	}
	return contentCh
}

func (c *s3Client) listInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	bucket, object := c.url2BucketAndObject()
	switch {
	case bucket == "" && object == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     bucket.Err,
				}
				return
			}
			content := new(client.Content)
			content.Name = bucket.Data.Name
			content.Size = 0
			content.Time = bucket.Data.CreationDate
			content.Type = os.ModeDir
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	default:
		metadata, err := c.api.StatObject(bucket, object)
		switch err.(type) {
		case nil:
			content := new(client.Content)
			content.Name = metadata.Key
			content.Time = metadata.LastModified
			content.Size = metadata.Size
			content.Type = os.FileMode(0664)
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		default:
			for object := range c.api.ListObjects(bucket, object, false) {
				if object.Err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     object.Err,
					}
					return
				}
				content := new(client.Content)
				content.Name = object.Data.Key
				switch {
				case strings.HasSuffix(object.Data.Key, "/"):
					content.Time = time.Now()
					content.Type = os.ModeDir
				default:
					content.Size = object.Data.Size
					content.Time = object.Data.LastModified
					content.Type = os.FileMode(0664)
				}
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
		}
	}
}

func (c *s3Client) listRecursiveInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	bucket, object := c.url2BucketAndObject()
	switch {
	case bucket == "" && object == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     bucket.Err,
				}
				return
			}
			for object := range c.api.ListObjects(bucket.Data.Name, object, true) {
				if object.Err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     object.Err,
					}
					return
				}
				content := new(client.Content)
				content.Name = strings.TrimSuffix(c.hostURL.String(), "/") + "/" + object.Data.Key
				content.Size = object.Data.Size
				content.Time = object.Data.LastModified
				content.Type = os.FileMode(0664)
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
		}
	default:
		for object := range c.api.ListObjects(bucket, object, true) {
			if object.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     object.Err,
				}
				return
			}
			content := new(client.Content)
			content.Name = strings.TrimSuffix(c.hostURL.String(), "/") + "/" + object.Data.Key
			content.Size = object.Data.Size
			content.Time = object.Data.LastModified
			content.Type = os.FileMode(0664)
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	}
}
