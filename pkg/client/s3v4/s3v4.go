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

package s3v4

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/minio-go"
	"github.com/minio/minio-xl/pkg/probe"
)

type s3Client struct {
	api     minio.API
	hostURL *client.URL
}

// New returns an initialized s3Client structure. if debug use a internal trace transport
func New(config *client.Config) (client.Client, *probe.Error) {
	u := client.NewURL(config.HostURL)
	transport := http.DefaultTransport
	if config.Debug == true {
		transport = httptracer.GetNewTraceTransport(NewTrace(), http.DefaultTransport)
	}
	s3Conf := minio.Config{
		AccessKeyID:     config.AccessKeyID,
		SecretAccessKey: config.SecretAccessKey,
		Transport:       transport,
		Endpoint:        u.Scheme + u.SchemeSeparator + u.Host,
	}
	s3Conf.AccessKeyID = config.AccessKeyID
	s3Conf.SecretAccessKey = config.SecretAccessKey
	s3Conf.Transport = transport
	s3Conf.SetUserAgent(config.AppName, config.AppVersion, config.AppComments...)
	s3Conf.Endpoint = u.Scheme + u.SchemeSeparator + u.Host
	api, err := minio.New(s3Conf)
	if err != nil {
		return nil, probe.NewError(err)
	}
	return &s3Client{api: api, hostURL: u}, nil
}

// URL get url
func (c *s3Client) URL() *client.URL {
	return c.hostURL
}

// Get - get object
func (c *s3Client) Get(offset, length int64) (io.ReadCloser, int64, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	reader, metadata, err := c.api.GetPartialObject(bucket, object, offset, length)
	if err != nil {
		return nil, length, probe.NewError(err)
	}
	return reader, metadata.Size, nil
}

// Remove - remove object or bucket
func (c *s3Client) Remove() *probe.Error {
	bucket, object := c.url2BucketAndObject()
	var err error
	if object == "" {
		err = c.api.RemoveBucket(bucket)
	} else {
		err = c.api.RemoveObject(bucket, object)
	}
	return probe.NewError(err)
}

// RemoveIncompleteUpload - remove multiparts of an incomplete multipart upload
func (c *s3Client) RemoveIncompleteUpload() *probe.Error {
	bucket, object := c.url2BucketAndObject()
	ch := c.api.RemoveIncompleteUpload(bucket, object)
	err := <-ch
	return probe.NewError(err)
}

// Share - get a usable get object url to share
func (c *s3Client) ShareDownload(expires time.Duration) (string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	presignedURL, err := c.api.PresignedGetObject(bucket, object, expires)
	if err != nil {
		return "", probe.NewError(err)
	}
	return presignedURL, nil
}

func (c *s3Client) ShareUpload(recursive bool, expires time.Duration, contentType string) (map[string]string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	p := minio.NewPostPolicy()
	if err := p.SetExpires(time.Now().UTC().Add(expires)); err != nil {
		return nil, probe.NewError(err)
	}
	if strings.TrimSpace(contentType) != "" || contentType != "" {
		// No need to verify for error here, since we have stripped out spaces
		p.SetContentType(contentType)
	}
	if err := p.SetBucket(bucket); err != nil {
		return nil, probe.NewError(err)
	}
	if recursive {
		if err := p.SetKeyStartsWith(object); err != nil {
			return nil, probe.NewError(err)
		}
	} else {
		if err := p.SetKey(object); err != nil {
			return nil, probe.NewError(err)
		}
	}
	m, err := c.api.PresignedPostPolicy(p)
	return m, probe.NewError(err)
}

// Put - put object
func (c *s3Client) Put(size int64, data io.Reader) *probe.Error {
	// md5 is purposefully ignored since AmazonS3 does not return proper md5sum
	// for a multipart upload and there is no need to cross verify,
	// invidual parts are properly verified
	bucket, object := c.url2BucketAndObject()
	err := c.api.PutObject(bucket, object, "application/octet-stream", size, data)
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse != nil {
			if errResponse.Code == "MethodNotAllowed" {
				return probe.NewError(client.ObjectAlreadyExists{Object: object})
			}
		}
		return probe.NewError(err)
	}
	return nil
}

// MakeBucket - make a new bucket
func (c *s3Client) MakeBucket() *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return probe.NewError(client.InvalidQueryURL{URL: c.hostURL.String()})
	}
	if bucket == "" {
		return probe.NewError(errors.New("Bucket name is empty."))
	}

	err := c.api.MakeBucket(bucket, minio.BucketACL("private"))
	if err != nil {
		return probe.NewError(err)
	}
	return nil
}

// GetBucketAccess get canned acl on a bucket
func (c *s3Client) GetBucketAccess() (acl string, error *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return "", probe.NewError(client.InvalidQueryURL{URL: c.hostURL.String()})
	}
	bucketACL, err := c.api.GetBucketACL(bucket)
	if err != nil {
		return "", probe.NewError(err)
	}
	return bucketACL.String(), nil
}

// SetBucketAccess set canned acl on a bucket
func (c *s3Client) SetBucketAccess(acl string) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return probe.NewError(client.InvalidQueryURL{URL: c.hostURL.String()})
	}
	err := c.api.SetBucketACL(bucket, minio.BucketACL(acl))
	if err != nil {
		return probe.NewError(err)
	}
	return nil
}

// Stat - send a 'HEAD' on a bucket or object to get its metadata
func (c *s3Client) Stat() (*client.Content, *probe.Error) {
	objectMetadata := new(client.Content)
	bucket, object := c.url2BucketAndObject()
	switch {
	// valid case for s3/...
	case bucket == "" && object == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				return nil, probe.NewError(bucket.Err)
			}
		}
		return &client.Content{Type: os.ModeDir}, nil
	}
	if object != "" {
		metadata, err := c.api.StatObject(bucket, object)
		if err != nil {
			errResponse := minio.ToErrorResponse(err)
			if errResponse != nil {
				if errResponse.Code == "NoSuchKey" {
					for content := range c.List(false, false) {
						if content.Err != nil {
							return nil, content.Err.Trace()
						}
						content.Content.Type = os.ModeDir
						content.Content.Name = object
						content.Content.Size = 0
						return content.Content, nil
					}
				}
			}
			return nil, probe.NewError(err)
		}
		objectMetadata.Name = metadata.Key
		objectMetadata.Time = metadata.LastModified
		objectMetadata.Size = metadata.Size
		objectMetadata.Type = os.FileMode(0664)
		return objectMetadata, nil
	}
	err := c.api.BucketExists(bucket)
	if err != nil {
		return nil, probe.NewError(err)
	}
	bucketMetadata := new(client.Content)
	bucketMetadata.Name = bucket
	bucketMetadata.Type = os.ModeDir
	return bucketMetadata, nil
}

// url2BucketAndObject gives bucketName and objectName from URL path
func (c *s3Client) url2BucketAndObject() (bucketName, objectName string) {
	path := c.hostURL.Path
	// Convert any virtual host styled requests
	//
	// For the time being this check is introduced for S3,
	// if you have custom virtual styled hosts please. list them below
	match, _ := filepath.Match("*.s3*.amazonaws.com", c.hostURL.Host)
	switch {
	case match == true:
		hostSplits := strings.SplitN(c.hostURL.Host, ".", 2)
		path = string(c.hostURL.Separator) + hostSplits[0] + c.hostURL.Path
	}
	splits := strings.SplitN(path, string(c.hostURL.Separator), 3)
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
func (c *s3Client) List(recursive, incomplete bool) <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	if incomplete {
		if recursive {
			go c.listIncompleteRecursiveInRoutine(contentCh)
		} else {
			go c.listIncompleteInRoutine(contentCh)
		}
	} else {
		if recursive {
			go c.listRecursiveInRoutine(contentCh)
		} else {
			go c.listInRoutine(contentCh)
		}
	}
	return contentCh
}

func (c *s3Client) listIncompleteInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(bucket.Err),
				}
				return
			}
			content := new(client.Content)
			content.Name = bucket.Stat.Name
			content.Size = 0
			content.Time = bucket.Stat.CreationDate
			content.Type = os.ModeDir
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	default:
		for object := range c.api.ListIncompleteUploads(b, o, false) {
			if object.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(object.Err),
				}
				return
			}
			content := new(client.Content)
			normalizedPrefix := strings.TrimSuffix(o, string(c.hostURL.Separator)) + string(c.hostURL.Separator)
			normalizedKey := object.Stat.Key
			if normalizedPrefix != object.Stat.Key && strings.HasPrefix(object.Stat.Key, normalizedPrefix) {
				normalizedKey = strings.TrimPrefix(object.Stat.Key, normalizedPrefix)
			}
			content.Name = normalizedKey
			switch {
			case strings.HasSuffix(object.Stat.Key, string(c.hostURL.Separator)):
				content.Time = time.Now()
				content.Type = os.ModeDir
			default:
				content.Size = object.Stat.Size
				content.Time = object.Stat.Initiated
				content.Type = os.ModeTemporary
			}
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	}
}

func (c *s3Client) listIncompleteRecursiveInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(bucket.Err),
				}
				return
			}
			for object := range c.api.ListIncompleteUploads(bucket.Stat.Name, o, true) {
				if object.Err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     probe.NewError(object.Err),
					}
					return
				}
				content := new(client.Content)
				content.Name = filepath.Join(bucket.Stat.Name, object.Stat.Key)
				content.Size = object.Stat.Size
				content.Time = object.Stat.Initiated
				content.Type = os.ModeTemporary
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
		}
	default:
		for object := range c.api.ListIncompleteUploads(b, o, true) {
			if object.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(object.Err),
				}
				return
			}
			content := new(client.Content)
			normalizedKey := object.Stat.Key
			switch {
			case o == "":
				// if no prefix provided and also URL is not delimited then we add bucket back into object name
				if strings.LastIndex(c.hostURL.Path, string(c.hostURL.Separator)) == 0 {
					if c.hostURL.String()[:strings.LastIndex(c.hostURL.String(), string(c.hostURL.Separator))+1] != b {
						normalizedKey = filepath.Join(b, object.Stat.Key)
					}
				}
			default:
				if strings.HasSuffix(o, string(c.hostURL.Separator)) {
					normalizedKey = strings.TrimPrefix(object.Stat.Key, o)
				}
			}
			content.Name = normalizedKey
			content.Size = object.Stat.Size
			content.Time = object.Stat.Initiated
			content.Type = os.ModeTemporary
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	}
}

func (c *s3Client) listInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(bucket.Err),
				}
				return
			}
			content := new(client.Content)
			content.Name = bucket.Stat.Name
			content.Size = 0
			content.Time = bucket.Stat.CreationDate
			content.Type = os.ModeDir
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	default:
		metadata, err := c.api.StatObject(b, o)
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
			for object := range c.api.ListObjects(b, o, false) {
				if object.Err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     probe.NewError(object.Err),
					}
					return
				}
				content := new(client.Content)
				normalizedPrefix := strings.TrimSuffix(o, string(c.hostURL.Separator)) + string(c.hostURL.Separator)
				normalizedKey := object.Stat.Key
				if normalizedPrefix != object.Stat.Key && strings.HasPrefix(object.Stat.Key, normalizedPrefix) {
					normalizedKey = strings.TrimPrefix(object.Stat.Key, normalizedPrefix)
				}
				content.Name = normalizedKey
				switch {
				case strings.HasSuffix(object.Stat.Key, string(c.hostURL.Separator)):
					content.Time = time.Now()
					content.Type = os.ModeDir
				default:
					content.Size = object.Stat.Size
					content.Time = object.Stat.LastModified
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
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		for bucket := range c.api.ListBuckets() {
			if bucket.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(bucket.Err),
				}
				return
			}
			for object := range c.api.ListObjects(bucket.Stat.Name, o, true) {
				if object.Err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     probe.NewError(object.Err),
					}
					return
				}
				content := new(client.Content)
				content.Name = filepath.Join(bucket.Stat.Name, object.Stat.Key)
				content.Size = object.Stat.Size
				content.Time = object.Stat.LastModified
				content.Type = os.FileMode(0664)
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
		}
	default:
		for object := range c.api.ListObjects(b, o, true) {
			if object.Err != nil {
				contentCh <- client.ContentOnChannel{
					Content: nil,
					Err:     probe.NewError(object.Err),
				}
				return
			}
			content := new(client.Content)
			normalizedKey := object.Stat.Key
			switch {
			case o == "":
				// if no prefix provided and also URL is not delimited then we add bucket back into object name
				if strings.LastIndex(c.hostURL.Path, string(c.hostURL.Separator)) == 0 {
					if c.hostURL.String()[:strings.LastIndex(c.hostURL.String(), string(c.hostURL.Separator))+1] != b {
						normalizedKey = filepath.Join(b, object.Stat.Key)
					}
				}
			default:
				if strings.HasSuffix(o, string(c.hostURL.Separator)) {
					normalizedKey = strings.TrimPrefix(object.Stat.Key, o)
				}
			}
			content.Name = normalizedKey
			content.Size = object.Stat.Size
			content.Time = object.Stat.LastModified
			content.Type = os.FileMode(0664)
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	}
}
