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

package main

import (
	"errors"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"io/ioutil"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/minio-go"
	"github.com/minio/minio/pkg/probe"
)

// S3 client
type s3Client struct {
	mutex        *sync.Mutex
	targetURL    *clientURL
	api          *minio.Client
	virtualStyle bool
}

// newFactory encloses New function with client cache.
func newFactory() func(config *Config) (Client, *probe.Error) {
	clientCache := make(map[uint32]*minio.Client)
	mutex := &sync.Mutex{}

	// Return New function.
	return func(config *Config) (Client, *probe.Error) {
		// Creates a parsed URL.
		targetURL := newClientURL(config.HostURL)
		// By default enable HTTPs.
		secure := true
		if targetURL.Scheme == "http" {
			secure = false
		}

		// Instantiate s3
		s3Clnt := &s3Client{}
		// Allocate a new mutex.
		s3Clnt.mutex = new(sync.Mutex)
		// Save the target URL.
		s3Clnt.targetURL = targetURL

		// Save if target supports virtual host style.
		hostName := targetURL.Host
		s3Clnt.virtualStyle = isVirtualHostStyle(hostName)

		if s3Clnt.virtualStyle {
			// If Amazon URL replace it with 's3.amazonaws.com'
			if isAmazon(hostName) {
				hostName = "s3.amazonaws.com"
			}
			// If Google URL replace it with 'storage.googleapis.com'
			if isGoogle(hostName) {
				hostName = "storage.googleapis.com"
			}
		}

		// Generate a hash out of s3Conf.
		confHash := fnv.New32a()
		confHash.Write([]byte(hostName + config.AccessKey + config.SecretKey))
		confSum := confHash.Sum32()

		// Lookup previous cache by hash.
		mutex.Lock()
		defer mutex.Unlock()
		var api *minio.Client
		found := false
		if api, found = clientCache[confSum]; !found {
			// Not found. Instantiate a new minio
			var e error
			if strings.ToUpper(config.Signature) == "S3V2" {
				// if Signature version '2' use NewV2 directly.
				api, e = minio.NewV2(hostName, config.AccessKey, config.SecretKey, secure)
			} else {
				// if Signature version '4' use NewV4 directly.
				api, e = minio.NewV4(hostName, config.AccessKey, config.SecretKey, secure)
			}
			if e != nil {
				return nil, probe.NewError(e)
			}
			if config.Debug {
				transport := http.DefaultTransport
				if config.Signature == "S3v4" {
					transport = httptracer.GetNewTraceTransport(newTraceV4(), http.DefaultTransport)
				}
				if config.Signature == "S3v2" {
					transport = httptracer.GetNewTraceTransport(newTraceV2(), http.DefaultTransport)
				}
				// Set custom transport.
				api.SetCustomTransport(transport)
			}
			// Cache the new minio client with hash of config as key.
			clientCache[confSum] = api
		}
		// Set app info.
		api.SetAppInfo(config.AppName, config.AppVersion)

		// Store the new api object.
		s3Clnt.api = api

		return s3Clnt, nil
	}
}

// s3New returns an initialized s3Client structure. If debug is enabled,
// it also enables an internal trace transport.
var s3New = newFactory()

// GetURL get url.
func (c *s3Client) GetURL() clientURL {
	return *c.targetURL
}

// Get - get object.
func (c *s3Client) Get() (io.Reader, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	reader, e := c.api.GetObject(bucket, object)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "AccessDenied" {
			return nil, probe.NewError(PathInsufficientPermission{Path: c.targetURL.String()})
		}
		if errResponse.Code == "NoSuchBucket" {
			return nil, probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return nil, probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return nil, probe.NewError(ObjectMissing{})
		}
		return nil, probe.NewError(e)
	}
	return reader, nil
}

// Copy - copy object
func (c *s3Client) Copy(source string, size int64, progress io.Reader) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return probe.NewError(BucketNameEmpty{})
	}
	// Empty copy conditions
	copyConds := minio.NewCopyConditions()
	e := c.api.CopyObject(bucket, object, source, copyConds)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "AccessDenied" {
			return probe.NewError(PathInsufficientPermission{
				Path: c.targetURL.String(),
			})
		}
		if errResponse.Code == "NoSuchBucket" {
			return probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return probe.NewError(ObjectMissing{})
		}
		return probe.NewError(e)
	}
	// Successful copy update progress bar if there is one.
	if progress != nil {
		if _, e := io.CopyN(ioutil.Discard, progress, size); e != nil {
			return probe.NewError(e)
		}
	}
	return nil
}

// Put - put object.
func (c *s3Client) Put(reader io.Reader, size int64, contentType string, progress io.Reader) (int64, *probe.Error) {
	// md5 is purposefully ignored since AmazonS3 does not return proper md5sum
	// for a multipart upload and there is no need to cross verify,
	// invidual parts are properly verified fully in transit and also upon completion
	// of the multipart request.
	bucket, object := c.url2BucketAndObject()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if bucket == "" {
		return 0, probe.NewError(BucketNameEmpty{})
	}
	n, e := c.api.PutObjectWithProgress(bucket, object, reader, contentType, progress)
	if e != nil {
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "UnexpectedEOF" || e == io.EOF {
			return n, probe.NewError(UnexpectedEOF{
				TotalSize:    size,
				TotalWritten: n,
			})
		}
		if errResponse.Code == "AccessDenied" {
			return n, probe.NewError(PathInsufficientPermission{
				Path: c.targetURL.String(),
			})
		}
		if errResponse.Code == "MethodNotAllowed" {
			return n, probe.NewError(ObjectAlreadyExists{
				Object: object,
			})
		}
		if errResponse.Code == "NoSuchBucket" {
			return n, probe.NewError(BucketDoesNotExist{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "InvalidBucketName" {
			return n, probe.NewError(BucketInvalid{
				Bucket: bucket,
			})
		}
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "InvalidArgument" {
			return n, probe.NewError(ObjectMissing{})
		}
		return n, probe.NewError(e)
	}
	return n, nil
}

// Remove - remove object or bucket.
func (c *s3Client) Remove(incomplete bool) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	// Remove only incomplete object.
	if incomplete && object != "" {
		e := c.api.RemoveIncompleteUpload(bucket, object)
		return probe.NewError(e)
	}
	var e error
	if object == "" {
		e = c.api.RemoveBucket(bucket)
	} else {
		e = c.api.RemoveObject(bucket, object)
	}
	return probe.NewError(e)
}

// We support '.' with bucket names but we fallback to using path
// style requests instead for such buckets
var validBucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`)

// isValidBucketName - verify bucket name in accordance with
//  - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html
func isValidBucketName(bucketName string) *probe.Error {
	if strings.TrimSpace(bucketName) == "" {
		return probe.NewError(errors.New("Bucket name cannot be empty."))
	}
	if len(bucketName) < 3 || len(bucketName) > 63 {
		return probe.NewError(errors.New("Bucket name should be more than 3 characters and less than 64 characters"))
	}
	if !validBucketName.MatchString(bucketName) {
		return probe.NewError(errors.New("Bucket name can contain alphabet, '-' and numbers, but first character should be an alphabet or number"))
	}
	return nil
}

// MakeBucket - make a new bucket.
func (c *s3Client) MakeBucket(region string) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if object != "" {
		return probe.NewError(BucketNameTopLevel{})
	}
	if err := isValidBucketName(bucket); err != nil {
		return err.Trace(bucket)
	}
	e := c.api.MakeBucket(bucket, region)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// GetAccess get access policy permissions.
func (c *s3Client) GetAccess() (string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return "", probe.NewError(BucketNameEmpty{})
	}
	bucketPolicy, e := c.api.GetBucketPolicy(bucket, object)
	if e != nil {
		return "", probe.NewError(e)
	}
	return string(bucketPolicy), nil
}

// SetAccess set access policy permissions.
func (c *s3Client) SetAccess(bucketPolicy string) *probe.Error {
	bucket, object := c.url2BucketAndObject()
	if bucket == "" {
		return probe.NewError(BucketNameEmpty{})
	}
	e := c.api.SetBucketPolicy(bucket, object, minio.BucketPolicy(bucketPolicy))
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Stat - send a 'HEAD' on a bucket or object to fetch its metadata.
func (c *s3Client) Stat() (*clientContent, *probe.Error) {
	c.mutex.Lock()
	objectMetadata := &clientContent{}
	bucket, object := c.url2BucketAndObject()
	// Bucket name cannot be empty, stat on URL has no meaning.
	if bucket == "" {
		c.mutex.Unlock()
		return nil, probe.NewError(BucketNameEmpty{})
	} else if object == "" {
		e := c.api.BucketExists(bucket)
		if e != nil {
			c.mutex.Unlock()
			return nil, probe.NewError(e)
		}
		bucketMetadata := &clientContent{}
		bucketMetadata.URL = *c.targetURL
		bucketMetadata.Type = os.ModeDir
		c.mutex.Unlock()
		return bucketMetadata, nil
	}
	metadata, e := c.api.StatObject(bucket, object)
	if e != nil {
		c.mutex.Unlock()
		errResponse := minio.ToErrorResponse(e)
		if errResponse.Code == "NoSuchKey" {
			// Append "/" to the object name proactively and see if the Listing
			// produces an output. If yes, then we treat it as a directory.
			prefixName := object
			// Trim any trailing separators and add it.
			prefixName = strings.TrimSuffix(prefixName, string(c.targetURL.Separator)) + string(c.targetURL.Separator)
			isRecursive := false
			for objectStat := range c.api.ListObjects(bucket, prefixName, isRecursive, nil) {
				if objectStat.Err != nil {
					return nil, probe.NewError(objectStat.Err)
				}
				content := clientContent{}
				content.URL = *c.targetURL
				content.Type = os.ModeDir
				return &content, nil
			}
			return nil, probe.NewError(PathNotFound{Path: c.targetURL.Path})
		}
		return nil, probe.NewError(e)
	}
	objectMetadata.URL = *c.targetURL
	objectMetadata.Time = metadata.LastModified
	objectMetadata.Size = metadata.Size
	objectMetadata.Type = os.FileMode(0664)
	c.mutex.Unlock()
	return objectMetadata, nil
}

func isAmazon(host string) bool {
	matchAmazon, _ := filepath.Match("*.s3*.amazonaws.com", host)
	return matchAmazon
}

func isGoogle(host string) bool {
	matchGoogle, _ := filepath.Match("*.storage.googleapis.com", host)
	return matchGoogle
}

// Figure out if the URL is of 'virtual host' style.
// Currently only supported hosts with virtual style are Amazon S3 and Google Cloud Storage.
func isVirtualHostStyle(host string) bool {
	return isAmazon(host) || isGoogle(host)
}

// url2BucketAndObject gives bucketName and objectName from URL path.
func (c *s3Client) url2BucketAndObject() (bucketName, objectName string) {
	path := c.targetURL.Path
	// Convert any virtual host styled requests.
	//
	// For the time being this check is introduced for S3,
	// If you have custom virtual styled hosts please.
	// List them below.
	if c.virtualStyle {
		var bucket string
		hostIndex := strings.Index(c.targetURL.Host, "s3")
		if hostIndex == -1 {
			hostIndex = strings.Index(c.targetURL.Host, "storage.googleapis")
		}
		if hostIndex > 0 {
			bucket = c.targetURL.Host[:hostIndex-1]
			path = string(c.targetURL.Separator) + bucket + c.targetURL.Path
		}
	}
	splits := strings.SplitN(path, string(c.targetURL.Separator), 3)
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

/// Bucket API operations.

// List - list at delimited path, if not recursive.
func (c *s3Client) List(recursive, incomplete bool) <-chan *clientContent {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	contentCh := make(chan *clientContent)
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

func (c *s3Client) listIncompleteInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, err := c.api.ListBuckets()
		if err != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(err),
			}
			return
		}
		isRecursive := false
		for _, bucket := range buckets {
			for object := range c.api.ListIncompleteUploads(bucket.Name, o, isRecursive, nil) {
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					return
				}
				content := &clientContent{}
				url := *c.targetURL
				// Join bucket with - incoming object key.
				url.Path = filepath.Join(string(url.Separator), bucket.Name, object.Key)
				if c.virtualStyle {
					url.Path = filepath.Join(string(url.Separator), object.Key)
				}
				switch {
				case strings.HasSuffix(object.Key, string(c.targetURL.Separator)):
					// We need to keep the trailing Separator, do not use filepath.Join().
					content.URL = url
					content.Time = time.Now()
					content.Type = os.ModeDir
				default:
					content.URL = url
					content.Size = object.Size
					content.Time = object.Initiated
					content.Type = os.ModeTemporary
				}
				contentCh <- content
			}
		}
	default:
		isRecursive := false
		for object := range c.api.ListIncompleteUploads(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				return
			}
			content := &clientContent{}
			url := *c.targetURL
			// Join bucket with - incoming object key.
			url.Path = filepath.Join(string(url.Separator), b, object.Key)
			if c.virtualStyle {
				url.Path = filepath.Join(string(url.Separator), object.Key)
			}
			switch {
			case strings.HasSuffix(object.Key, string(c.targetURL.Separator)):
				// We need to keep the trailing Separator, do not use filepath.Join().
				content.URL = url
				content.Time = time.Now()
				content.Type = os.ModeDir
			default:
				content.URL = url
				content.Size = object.Size
				content.Time = object.Initiated
				content.Type = os.ModeTemporary
			}
			contentCh <- content
		}
	}
}

func (c *s3Client) listIncompleteRecursiveInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, err := c.api.ListBuckets()
		if err != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(err),
			}
			return
		}
		isRecursive := true
		for _, bucket := range buckets {
			for object := range c.api.ListIncompleteUploads(bucket.Name, o, isRecursive, nil) {
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					return
				}
				url := *c.targetURL
				url.Path = filepath.Join(url.Path, bucket.Name, object.Key)
				content := &clientContent{}
				content.URL = url
				content.Size = object.Size
				content.Time = object.Initiated
				content.Type = os.ModeTemporary
				contentCh <- content
			}
		}
	default:
		isRecursive := true
		for object := range c.api.ListIncompleteUploads(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				return
			}
			url := *c.targetURL
			// Join bucket and incoming object key.
			url.Path = filepath.Join(string(url.Separator), b, object.Key)
			if c.virtualStyle {
				url.Path = filepath.Join(string(url.Separator), object.Key)
			}
			content := &clientContent{}
			content.URL = url
			content.Size = object.Size
			content.Time = object.Initiated
			content.Type = os.ModeTemporary
			contentCh <- content
		}
	}
}

func (c *s3Client) listInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, e := c.api.ListBuckets()
		if e != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(e),
			}
			return
		}
		for _, bucket := range buckets {
			url := *c.targetURL
			url.Path = filepath.Join(url.Path, bucket.Name)
			content := &clientContent{}
			content.URL = url
			content.Size = 0
			content.Time = bucket.CreationDate
			content.Type = os.ModeDir
			contentCh <- content
		}
	case b != "" && !strings.HasSuffix(c.targetURL.Path, string(c.targetURL.Separator)) && o == "":
		buckets, e := c.api.ListBuckets()
		if e != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(e),
			}
		}
		for _, bucket := range buckets {
			if bucket.Name == b {
				content := &clientContent{}
				content.URL = *c.targetURL
				content.Size = 0
				content.Time = bucket.CreationDate
				content.Type = os.ModeDir
				contentCh <- content
				break
			}
		}
	default:
		metadata, e := c.api.StatObject(b, o)
		switch e.(type) {
		case nil:
			content := &clientContent{}
			content.URL = *c.targetURL
			content.Time = metadata.LastModified
			content.Size = metadata.Size
			content.Type = os.FileMode(0664)
			contentCh <- content
		default:
			isRecursive := false
			for object := range c.api.ListObjects(b, o, isRecursive, nil) {
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					return
				}
				content := &clientContent{}
				url := *c.targetURL
				// Join bucket and incoming object key.
				url.Path = filepath.Join(string(url.Separator), b, object.Key)
				if c.virtualStyle {
					url.Path = filepath.Join(string(url.Separator), object.Key)
				}
				switch {
				case strings.HasSuffix(object.Key, string(c.targetURL.Separator)):
					// We need to keep the trailing Separator, do not use filepath.Join().
					content.URL = url
					content.Time = time.Now()
					content.Type = os.ModeDir
				default:
					content.URL = url
					content.Size = object.Size
					content.Time = object.LastModified
					content.Type = os.FileMode(0664)
				}
				contentCh <- content
			}
		}
	}
}

func (c *s3Client) listRecursiveInRoutine(contentCh chan *clientContent) {
	defer close(contentCh)
	// get bucket and object from URL.
	b, o := c.url2BucketAndObject()
	switch {
	case b == "" && o == "":
		buckets, err := c.api.ListBuckets()
		if err != nil {
			contentCh <- &clientContent{
				Err: probe.NewError(err),
			}
			return
		}
		for _, bucket := range buckets {
			bucketURL := *c.targetURL
			bucketURL.Path = filepath.Join(bucketURL.Path, bucket.Name)
			contentCh <- &clientContent{
				URL:  bucketURL,
				Type: os.ModeDir,
				Time: bucket.CreationDate,
			}
			isRecursive := true
			for object := range c.api.ListObjects(bucket.Name, o, isRecursive, nil) {
				if object.Err != nil {
					contentCh <- &clientContent{
						Err: probe.NewError(object.Err),
					}
					continue
				}
				content := &clientContent{}
				objectURL := *c.targetURL
				objectURL.Path = filepath.Join(objectURL.Path, bucket.Name, object.Key)
				content.URL = objectURL
				content.Size = object.Size
				content.Time = object.LastModified
				content.Type = os.FileMode(0664)
				contentCh <- content
			}
		}
	default:
		isRecursive := true
		for object := range c.api.ListObjects(b, o, isRecursive, nil) {
			if object.Err != nil {
				contentCh <- &clientContent{
					Err: probe.NewError(object.Err),
				}
				continue
			}
			content := &clientContent{}
			url := *c.targetURL
			// Join bucket and incoming object key.
			url.Path = filepath.Join(string(url.Separator), b, object.Key)
			// If virtualStyle replace the url.Path back.
			if c.virtualStyle {
				url.Path = filepath.Join(string(url.Separator), object.Key)
			}
			content.URL = url
			content.Size = object.Size
			content.Time = object.LastModified
			content.Type = os.FileMode(0664)
			contentCh <- content
		}
	}
}

// ShareDownload - get a usable presigned object url to share.
func (c *s3Client) ShareDownload(expires time.Duration) (string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	// No additional request parameters are set for the time being.
	reqParams := make(url.Values)
	presignedURL, e := c.api.PresignedGetObject(bucket, object, expires, reqParams)
	if e != nil {
		return "", probe.NewError(e)
	}
	return presignedURL.String(), nil
}

// ShareUpload - get data for presigned post http form upload.
func (c *s3Client) ShareUpload(isRecursive bool, expires time.Duration, contentType string) (map[string]string, *probe.Error) {
	bucket, object := c.url2BucketAndObject()
	p := minio.NewPostPolicy()
	if e := p.SetExpires(time.Now().UTC().Add(expires)); e != nil {
		return nil, probe.NewError(e)
	}
	if strings.TrimSpace(contentType) != "" || contentType != "" {
		// No need to verify for error here, since we have stripped out spaces.
		p.SetContentType(contentType)
	}
	if e := p.SetBucket(bucket); e != nil {
		return nil, probe.NewError(e)
	}
	if isRecursive {
		if e := p.SetKeyStartsWith(object); e != nil {
			return nil, probe.NewError(e)
		}
	} else {
		if e := p.SetKey(object); e != nil {
			return nil, probe.NewError(e)
		}
	}
	_, m, e := c.api.PresignedPostPolicy(p)
	return m, probe.NewError(e)
}
