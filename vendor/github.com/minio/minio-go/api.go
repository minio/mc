/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package minio

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// s3 region map used by bucket location constraint if necessary.
var regions = map[string]string{
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

	// Add google cloud storage as one of the regions
	"storage.googleapis.com": "google",
}

// getRegion returns a region based on its endpoint mapping.
func getRegion(host string) (region string) {
	if _, ok := regions[host]; ok {
		return regions[host]
	}
	// Region cannot be empty according to Amazon S3 for AWS Signature Version 4.
	return "us-east-1"
}

// getEndpoint returns a endpoint based on its region.
func getEndpoint(region string) (endpoint string) {
	for h, r := range regions {
		if r == region {
			return h
		}
	}
	return "s3.amazonaws.com"
}

// SignatureType is type of Authorization requested for a given HTTP request.
type SignatureType int

// Different types of supported signatures - default is Latest i.e SignatureV4.
const (
	Latest SignatureType = iota
	SignatureV4
	SignatureV2
)

// isV2 - is signature SignatureV2?
func (s SignatureType) isV2() bool {
	return s == SignatureV2
}

// isV4 - is signature SignatureV4?
func (s SignatureType) isV4() bool {
	return s == SignatureV4
}

// isLatest - is signature Latest?
func (s SignatureType) isLatest() bool {
	return s == Latest
}

// Config - main configuration struct used to set endpoint, credentials, and other options for requests.
type Config struct {
	///  Standard options
	AccessKeyID     string        // AccessKeyID required for authorized requests.
	SecretAccessKey string        // SecretAccessKey required for authorized requests.
	Endpoint        string        // host endpoint eg:- https://s3.amazonaws.com
	Signature       SignatureType // choose a signature type if necessary.

	/// Advanced options
	// Optional field. If empty, region is determined automatically.
	// Set to override default behavior.
	Region string

	/// Really Advanced options
	//
	// Set this to override default transport ``http.DefaultTransport``
	//
	// This transport is usually needed for debugging OR to add your own
	// custom TLS certificates on the client transport, for custom CA's and
	// certs which are not part of standard certificate authority follow this
	// example:-
	//
	//   tr := &http.Transport{
	//           TLSClientConfig:    &tls.Config{RootCAs: pool},
	//           DisableCompression: true,
	//   }
	//
	Transport http.RoundTripper

	/// Internal options
	// use SetUserAgent append to default, useful when minio-go is used with in your application
	userAgent            string
	isUserAgentSet       bool // allow user agent's to be set only once
	isVirtualHostedStyle bool // set when virtual hostnames are on
}

// Global constants
const (
	LibraryName    = "minio-go"
	LibraryVersion = "0.2.5"
)

// isAnonymous - True if config doesn't have access and secret keys.
func (c *Config) isAnonymous() bool {
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		return false
	}
	return true
}

// setBucketRegion fetches the region and updates config,
// additionally it also constructs a proper endpoint based on that region.
func (c *Config) setBucketRegion() {
	u, err := url.Parse(c.Endpoint)
	if err != nil {
		return
	}

	if !c.isVirtualHostedStyle {
		c.Region = getRegion(u.Host)
		return
	}

	var bucket, host string
	hostIndex := strings.Index(u.Host, "s3")
	if hostIndex == -1 {
		hostIndex = strings.Index(u.Host, "storage.googleapis.com")
	}
	if hostIndex > 0 {
		host = u.Host[hostIndex:]
		bucket = u.Host[:hostIndex-1]
	}

	genericGoogle, _ := filepath.Match("*.storage.googleapis.com", u.Host)
	if genericGoogle {
		// returning standard region for google for now, can be changed in future
		// to query for region in case it is useful
		c.Region = getRegion(host)
		return
	}
	genericS3, _ := filepath.Match("*.s3.amazonaws.com", u.Host)
	if !genericS3 {
		c.Region = getRegion(host)
		return
	}

	// query aws s3 for the region for case of bucketName.s3.amazonaws.com
	u.Host = host
	tempConfig := Config{}
	tempConfig.AccessKeyID = c.AccessKeyID
	tempConfig.SecretAccessKey = c.SecretAccessKey
	tempConfig.Endpoint = u.String()
	tempConfig.Region = getRegion(u.Host)
	tempConfig.isVirtualHostedStyle = false
	s3API := API{s3API{&tempConfig}}
	region, err := s3API.getBucketLocation(bucket)
	if err != nil {
		c.Region = getRegion(host)
		return
	}
	// if region returned from getBucketLocation is null
	// and if genericS3 is enabled - set back to 'us-east-1'.
	if region == "" {
		if genericS3 {
			region = "us-east-1"
		}
	}
	c.Region = region
	c.setEndpoint(region, bucket, u.Scheme)
	return
}

// setEndpoint - construct final endpoint based on region, bucket and scheme
func (c *Config) setEndpoint(region, bucket, scheme string) {
	var host string
	for k, v := range regions {
		if region == v {
			host = k
		}
	}
	// construct the new URL endpoint based on the region.
	newURL := new(url.URL)
	newURL.Host = bucket + "." + host
	newURL.Scheme = scheme
	c.Endpoint = newURL.String()
	return
}

// SetUserAgent - append to a default user agent
func (c *Config) SetUserAgent(name string, version string, comments ...string) {
	if c.isUserAgentSet {
		// if user agent already set do not set it
		return
	}
	// if no name and version is set we do not add new user agents
	if name != "" && version != "" {
		c.userAgent = c.userAgent + " " + name + "/" + version + " (" + strings.Join(comments, "; ") + ") "
		c.isUserAgentSet = true
	}
}

// API is a container which delegates methods that comply with CloudStorageAPI interface.
type API struct {
	s3API
}

func isVirtualHostedStyle(host string) bool {
	isS3VirtualHost, _ := filepath.Match("*.s3*.amazonaws.com", host)
	isGoogleVirtualHost, _ := filepath.Match("*.storage.googleapis.com", host)
	return isS3VirtualHost || isGoogleVirtualHost
}

// New - instantiate minio client API with your input Config{}.
func New(config Config) (CloudStorageAPI, error) {
	u, err := url.Parse(config.Endpoint)
	if err != nil {
		return API{}, err
	}
	config.isVirtualHostedStyle = isVirtualHostedStyle(u.Host)
	// if not region is set, procure it from getBucketRegion if possible.
	if config.Region == "" {
		config.setBucketRegion()
	}
	/// Google cloud storage should be set to signature V2, force it if not.
	if config.Region == "google" && config.Signature != SignatureV2 {
		config.Signature = SignatureV2
	}
	config.SetUserAgent(LibraryName, LibraryVersion, runtime.GOOS, runtime.GOARCH)
	config.isUserAgentSet = false // default
	return API{s3API{&config}}, nil
}

// PresignedPostPolicy return POST form data that can be used for object upload.
func (a API) PresignedPostPolicy(p *PostPolicy) (map[string]string, error) {
	if p.expiration.IsZero() {
		return nil, errors.New("Expiration time must be specified")
	}
	if _, ok := p.formData["key"]; !ok {
		return nil, errors.New("object key must be specified")
	}
	if _, ok := p.formData["bucket"]; !ok {
		return nil, errors.New("bucket name must be specified")
	}
	return a.presignedPostPolicy(p), nil
}

/// Object operations.

// PresignedPutObject get a presigned URL to upload an object.
// Expires maximum is 7days - ie. 604800 and minimum is 1.
func (a API) PresignedPutObject(bucket, object string, expires time.Duration) (string, error) {
	expireSeconds := int64(expires / time.Second)
	if expireSeconds < 1 || expireSeconds > 604800 {
		return "", invalidArgumentError("")
	}
	return a.presignedPutObject(bucket, object, expireSeconds)
}

// PresignedGetObject get a presigned URL to retrieve an object for third party apps.
func (a API) PresignedGetObject(bucket, object string, expires time.Duration) (string, error) {
	expireSeconds := int64(expires / time.Second)
	if expireSeconds < 1 || expireSeconds > 604800 {
		return "", invalidArgumentError("")
	}
	return a.presignedGetObject(bucket, object, expireSeconds, 0, 0)
}

// GetObject retrieve object. retrieves full object, if you need ranges use GetPartialObject.
func (a API) GetObject(bucket, object string) (io.ReadSeeker, error) {
	if err := invalidBucketError(bucket); err != nil {
		return nil, err
	}
	if err := invalidObjectError(object); err != nil {
		return nil, err
	}
	// get object
	return newObjectReadSeeker(a, bucket, object), nil
}

// GetPartialObject retrieve partial object.
//
// Takes range arguments to download the specified range bytes of an object.
// Setting offset and length = 0 will download the full object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a API) GetPartialObject(bucket, object string, offset, length int64) (io.ReadSeeker, error) {
	if err := invalidBucketError(bucket); err != nil {
		return nil, err
	}
	if err := invalidObjectError(object); err != nil {
		return nil, err
	}
	// get partial object.
	return newObjectReadSeeker(a, bucket, object), nil
}

// completedParts is a wrapper to make parts sortable by their part numbers.
// multi part completion requires list of multi parts to be sorted.
type completedParts []completePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

// minimumPartSize minimum part size per object after which PutObject behaves internally as multipart.
var minimumPartSize int64 = 1024 * 1024 * 5

// maxParts - maximum parts for a single multipart session.
var maxParts = int64(10000)

// maxPartSize - maximum part size for a single multipart upload operation.
var maxPartSize int64 = 1024 * 1024 * 1024 * 5

// maxConcurrentQueue - max concurrent upload queue.
var maxConcurrentQueue int64 = 4

// calculatePartSize - calculate the optimal part size for the given objectSize.
//
// NOTE: Assumption here is that for any given object upload to a S3 compatible object
// storage it will have the following parameters as constants.
//
//  maxParts
//  maximumPartSize
//  minimumPartSize
//
// if the partSize after division with maxParts is greater than minimumPartSize
// then choose miniumPartSize as the new part size, if not return minimumPartSize.
//
// special case where it happens to be that partSize is indeed bigger than the
// maximum part size just return maxPartSize.
func calculatePartSize(objectSize int64) int64 {
	// if object size is -1 choose part size as 5GB.
	if objectSize == -1 {
		return maxPartSize
	}
	// make sure last part has enough buffer and handle this poperly.
	partSize := (objectSize / (maxParts - 1))
	if partSize > minimumPartSize {
		if partSize > maxPartSize {
			return maxPartSize
		}
		return partSize
	}
	return minimumPartSize
}

// Initiate a fresh multipart upload
func (a API) newObjectUpload(bucket, object, contentType string, size int64, data io.ReadSeeker) error {
	// Initiate a new multipart upload request.
	initMultipartUploadResult, err := a.initiateMultipartUpload(bucket, object)
	if err != nil {
		return err
	}
	uploadID := initMultipartUploadResult.UploadID
	complMultipartUpload := completeMultipartUpload{}

	// Calculate optimal part size for a given size.
	partSize := calculatePartSize(size)
	// Allocate bufferred error channel for maximum parts.
	errCh := make(chan error, maxParts)
	// Limit multi part queue size to concurrent.
	mpQueueCh := make(chan struct{}, maxConcurrentQueue)
	defer close(errCh)
	defer close(mpQueueCh)
	// Allocate a new wait group
	wg := new(sync.WaitGroup)

	partNumber := 1
	var isEnableSha256Sum bool
	if a.config.Signature.isV4() {
		isEnableSha256Sum = true
	}
	for part := range partsManager(data, partSize, isEnableSha256Sum) {
		// Limit to 4 parts a given time.
		mpQueueCh <- struct{}{}
		// Account for all parts uploaded simultaneousy.
		wg.Add(1)
		part.Number = partNumber
		go func(errCh chan<- error, mpQueueCh <-chan struct{}, part partMetadata) {
			defer wg.Done()
			defer func() {
				<-mpQueueCh
			}()
			if part.Err != nil {
				errCh <- part.Err
				return
			}
			var complPart completePart
			complPart, err = a.uploadPart(bucket, object, uploadID, part)
			if err != nil {
				errCh <- err
				return
			}
			complMultipartUpload.Parts = append(complMultipartUpload.Parts, complPart)
			errCh <- nil
		}(errCh, mpQueueCh, part)
		partNumber++
	}
	wg.Wait()
	if err := <-errCh; err != nil {
		return err
	}
	sort.Sort(completedParts(complMultipartUpload.Parts))
	_, err = a.completeMultipartUpload(bucket, object, uploadID, complMultipartUpload)
	if err != nil {
		return err
	}
	return nil
}

func (a API) listObjectPartsRecursive(bucket, object, uploadID string) <-chan objectPartMetadata {
	objectPartCh := make(chan objectPartMetadata, 1000)
	go a.listObjectPartsRecursiveInRoutine(bucket, object, uploadID, objectPartCh)
	return objectPartCh
}

func (a API) listObjectPartsRecursiveInRoutine(bucket, object, uploadID string, ch chan<- objectPartMetadata) {
	defer close(ch)
	listObjPartsResult, err := a.listObjectParts(bucket, object, uploadID, 0, 1000)
	if err != nil {
		ch <- objectPartMetadata{
			Err: err,
		}
		return
	}
	for _, uploadedObjectPart := range listObjPartsResult.ObjectParts {
		ch <- uploadedObjectPart
	}
	for {
		if !listObjPartsResult.IsTruncated {
			break
		}
		nextPartNumberMarker := listObjPartsResult.NextPartNumberMarker
		listObjPartsResult, err = a.listObjectParts(bucket, object, uploadID, nextPartNumberMarker, 1000)
		if err != nil {
			ch <- objectPartMetadata{
				Err: err,
			}
			return
		}
		for _, uploadedObjectPart := range listObjPartsResult.ObjectParts {
			ch <- uploadedObjectPart
		}
	}
}

// getTotalMultipartSize - calculate total uploaded size for the a given multipart object.
func (a API) getTotalMultipartSize(bucket, object, uploadID string) (int64, error) {
	var size int64
	for part := range a.listObjectPartsRecursive(bucket, object, uploadID) {
		if part.Err != nil {
			return 0, part.Err
		}
		size += part.Size
	}
	return size, nil
}

// continue previously interrupted multipart upload object at `uploadID`
func (a API) continueObjectUpload(bucket, object, uploadID string, size int64, data io.ReadSeeker) error {
	var seekOffset int64
	partNumber := 1
	completeMultipartUpload := completeMultipartUpload{}
	for objPart := range a.listObjectPartsRecursive(bucket, object, uploadID) {
		if objPart.Err != nil {
			return objPart.Err
		}
		// partNumbers are sorted in listObjectParts.
		if partNumber != objPart.PartNumber {
			break
		}
		var completedPart completePart
		completedPart.PartNumber = objPart.PartNumber
		completedPart.ETag = objPart.ETag
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, completedPart)
		seekOffset += objPart.Size // Add seek Offset for future Seek to skip entries.
		partNumber++               // Update partNumber sequentially to verify and skip.
	}

	// Calculate the optimal part size for a given size.
	partSize := calculatePartSize(size)
	// Allocate bufferred error channel for maximum parts.
	errCh := make(chan error, maxParts)
	// Limit multipart queue size to maxConcurrentQueue.
	mpQueueCh := make(chan struct{}, maxConcurrentQueue)
	defer close(errCh)
	defer close(mpQueueCh)
	// Allocate a new wait group.
	wg := new(sync.WaitGroup)

	if _, err := data.Seek(seekOffset, 0); err != nil {
		return err
	}
	var isEnableSha256Sum bool
	if a.config.Signature.isV4() {
		isEnableSha256Sum = true
	}
	for part := range partsManager(data, partSize, isEnableSha256Sum) {
		// Limit to 4 parts a given time.
		mpQueueCh <- struct{}{}
		// Account for all parts uploaded simultaneousy.
		wg.Add(1)
		part.Number = partNumber
		go func(errCh chan<- error, mpQueueCh <-chan struct{}, part partMetadata) {
			defer wg.Done()
			defer func() {
				<-mpQueueCh
			}()
			if part.Err != nil {
				errCh <- part.Err
				return
			}
			complPart, err := a.uploadPart(bucket, object, uploadID, part)
			if err != nil {
				errCh <- err
				return
			}
			completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, complPart)
			errCh <- nil
		}(errCh, mpQueueCh, part)
		partNumber++
	}
	wg.Wait()
	if err := <-errCh; err != nil {
		return err
	}
	sort.Sort(completedParts(completeMultipartUpload.Parts))
	_, err := a.completeMultipartUpload(bucket, object, uploadID, completeMultipartUpload)
	if err != nil {
		return err
	}
	return nil
}

// PutObject create an object in a bucket.
//
// You must have WRITE permissions on a bucket to create an object.
//
//  - For size lesser than 5MB PutObject automatically does single Put operation.
//  - For size equal to 0Bytes PutObject automatically does single Put operation.
//  - For size larger than 5MB PutObject automatically does resumable multipart operation.
//  - For size input as -1 PutObject treats it as a stream and does multipart operation until
//    input stream reaches EOF. Maximum object size that can be uploaded through this operation
//    will be 5TB.
//
// NOTE: if you are using Google Cloud Storage. Then there is no resumable multipart
// upload support yet. Currently PutObject will behave like a single PUT operation and would
// only upload for file sizes upto maximum 5GB. (maximum limit for single PUT operation).
//
// For un-authenticated requests S3 doesn't allow multipart upload, so we fall back to single
// PUT operation.
func (a API) PutObject(bucket, object, contentType string, data io.ReadSeeker, size int64) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	if err := invalidArgumentError(object); err != nil {
		return err
	}
	// NOTE: S3 doesn't allow anonymous multipart requests.
	if strings.Contains(a.config.Endpoint, "amazonaws.com") || strings.Contains(a.config.Endpoint, "googleapis.com") {
		if a.config.isAnonymous() {
			if size == -1 {
				return ErrorResponse{
					Code:     "NotImplemented",
					Message:  "For Anonymous requests Content-Length cannot be '-1'.",
					Resource: separator + bucket + separator + object,
				}
			}
			if size > maxPartSize {
				return ErrorResponse{
					Code:     "EntityTooLarge",
					Message:  "Your proposed upload exceeds the maximum allowed object size '5GB' for single PUT operation.",
					Resource: separator + bucket + separator + object,
				}
			}
			// For anonymous requests, we will not calculate sha256 and md5sum.
			putObjMetadata := putObjectMetadata{
				MD5Sum:      nil,
				Sha256Sum:   nil,
				ReadCloser:  ioutil.NopCloser(data),
				Size:        size,
				ContentType: contentType,
			}
			_, err := a.putObject(bucket, object, putObjMetadata)
			if err != nil {
				return err
			}
		}
	}
	// Special handling just for Google Cloud Storage.
	// TODO - we should remove this in future when we fully implement Resumable object upload.
	if strings.Contains(a.config.Endpoint, "googleapis.com") {
		if size > maxPartSize {
			return ErrorResponse{
				Code:     "EntityTooLarge",
				Message:  "Your proposed upload exceeds the maximum allowed object size '5GB' for single PUT operation.",
				Resource: separator + bucket + separator + object,
			}
		}
		putObjMetadata := putObjectMetadata{
			MD5Sum:      nil,
			Sha256Sum:   nil,
			ReadCloser:  ioutil.NopCloser(data),
			Size:        size,
			ContentType: contentType,
		}
		// NOTE: with Google Cloud Storage, Content-MD5 is deliberately skipped.
		if _, err := a.putObject(bucket, object, putObjMetadata); err != nil {
			return err
		}
		return nil
	}
	switch {
	case size < minimumPartSize && size >= 0:
		dataBytes, err := ioutil.ReadAll(data)
		if err != nil {
			return err
		}
		if int64(len(dataBytes)) != size {
			return ErrorResponse{
				Code:     "UnexpectedShortRead",
				Message:  "Data read ‘" + strconv.FormatInt(int64(len(dataBytes)), 10) + "’ is not equal to expected size ‘" + strconv.FormatInt(size, 10) + "’",
				Resource: separator + bucket + separator + object,
			}
		}
		putObjMetadata := putObjectMetadata{
			MD5Sum:      sumMD5(dataBytes),
			Sha256Sum:   sum256(dataBytes),
			ReadCloser:  ioutil.NopCloser(bytes.NewReader(dataBytes)),
			Size:        size,
			ContentType: contentType,
		}
		// Single Part use case, use PutObject directly.
		_, err = a.putObject(bucket, object, putObjMetadata)
		if err != nil {
			return err
		}
		return nil
	case size >= minimumPartSize || size == -1:
		var inProgress bool
		var inProgressUploadID string
		for mpUpload := range a.listMultipartUploadsRecursive(bucket, object) {
			if mpUpload.Err != nil {
				return mpUpload.Err
			}
			if mpUpload.Key == object {
				inProgress = true
				inProgressUploadID = mpUpload.UploadID
				break
			}
		}
		if !inProgress {
			return a.newObjectUpload(bucket, object, contentType, size, data)
		}
		return a.continueObjectUpload(bucket, object, inProgressUploadID, size, data)
	}
	return errors.New("Unexpected control flow, please report this error at https://github.com/minio/minio-go/issues")
}

// StatObject verify if object exists and you have permission to access it.
func (a API) StatObject(bucket, object string) (ObjectStat, error) {
	if err := invalidBucketError(bucket); err != nil {
		return ObjectStat{}, err
	}
	if err := invalidObjectError(object); err != nil {
		return ObjectStat{}, err
	}
	return a.headObject(bucket, object)
}

// RemoveObject remove an object from a bucket.
func (a API) RemoveObject(bucket, object string) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	if err := invalidObjectError(object); err != nil {
		return err
	}
	return a.deleteObject(bucket, object)
}

/// Bucket operations

// MakeBucket makes a new bucket.
//
// Optional arguments are acl - by default all buckets are created
// with ``private`` acl.
//
// ACL valid values
//
//  private - owner gets full access [default].
//  public-read - owner gets full access, all others get read access.
//  public-read-write - owner gets full access, all others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
//
func (a API) MakeBucket(bucket string, acl BucketACL) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	if !acl.isValidBucketACL() {
		return invalidArgumentError("")
	}
	location := a.config.Region
	if location == "us-east-1" {
		location = ""
	}
	if location == "google" {
		location = ""
	}
	return a.putBucket(bucket, string(acl), location)
}

// SetBucketACL set the permissions on an existing bucket using access control lists (ACL).
//
// For example
//
//  private - owner gets full access [default].
//  public-read - owner gets full access, all others get read access.
//  public-read-write - owner gets full access, all others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
//
func (a API) SetBucketACL(bucket string, acl BucketACL) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	if !acl.isValidBucketACL() {
		return invalidArgumentError("")
	}
	return a.putBucketACL(bucket, string(acl))
}

// GetBucketACL get the permissions on an existing bucket.
//
// Returned values are:
//
//  private - owner gets full access.
//  public-read - owner gets full access, others get read access.
//  public-read-write - owner gets full access, others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
//
func (a API) GetBucketACL(bucket string) (BucketACL, error) {
	if err := invalidBucketError(bucket); err != nil {
		return "", err
	}
	policy, err := a.getBucketACL(bucket)
	if err != nil {
		return "", err
	}
	grants := policy.AccessControlList.Grant
	switch {
	case len(grants) == 1:
		if grants[0].Grantee.URI == "" && grants[0].Permission == "FULL_CONTROL" {
			return BucketACL("private"), nil
		}
	case len(grants) == 2:
		for _, g := range grants {
			if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AuthenticatedUsers" && g.Permission == "READ" {
				return BucketACL("authenticated-read"), nil
			}
			if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == "READ" {
				return BucketACL("public-read"), nil
			}
		}
	case len(grants) == 3:
		for _, g := range grants {
			if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == "WRITE" {
				return BucketACL("public-read-write"), nil
			}
		}
	}
	return "", ErrorResponse{
		Code:      "NoSuchBucketPolicy",
		Message:   "The specified bucket does not have a bucket policy.",
		Resource:  "/" + bucket,
		RequestID: "minio",
	}
}

// BucketExists verify if bucket exists and you have permission to access it.
func (a API) BucketExists(bucket string) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	return a.headBucket(bucket)
}

// RemoveBucket deletes the bucket named in the URI.
//
//  All objects (including all object versions and delete markers).
//  in the bucket must be deleted before successfully attempting this request.
func (a API) RemoveBucket(bucket string) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	return a.deleteBucket(bucket)
}

func (a API) listMultipartUploadsRecursive(bucket, object string) <-chan ObjectMultipartStat {
	ch := make(chan ObjectMultipartStat, 1000)
	go a.listMultipartUploadsRecursiveInRoutine(bucket, object, ch)
	return ch
}

func (a API) listMultipartUploadsRecursiveInRoutine(bucket, object string, ch chan<- ObjectMultipartStat) {
	defer close(ch)
	listMultipartUplResult, err := a.listMultipartUploads(bucket, "", "", object, "", 1000)
	if err != nil {
		ch <- ObjectMultipartStat{
			Err: err,
		}
		return
	}
	for _, multiPartUpload := range listMultipartUplResult.Uploads {
		ch <- multiPartUpload
	}
	for {
		if !listMultipartUplResult.IsTruncated {
			break
		}
		listMultipartUplResult, err = a.listMultipartUploads(bucket,
			listMultipartUplResult.NextKeyMarker, listMultipartUplResult.NextUploadIDMarker, object, "", 1000)
		if err != nil {
			ch <- ObjectMultipartStat{
				Err: err,
			}
			return
		}
		for _, multiPartUpload := range listMultipartUplResult.Uploads {
			ch <- multiPartUpload
		}
	}
}

// listIncompleteUploadsInRoutine is an internal goroutine function called for listing objects.
func (a API) listIncompleteUploadsInRoutine(bucket, prefix string, recursive bool, ch chan<- ObjectMultipartStat) {
	defer close(ch)
	if err := invalidBucketError(bucket); err != nil {
		ch <- ObjectMultipartStat{
			Err: err,
		}
		return
	}
	switch {
	case recursive == true:
		var multipartMarker string
		var uploadIDMarker string
		for {
			result, err := a.listMultipartUploads(bucket, multipartMarker, uploadIDMarker, prefix, "", 1000)
			if err != nil {
				ch <- ObjectMultipartStat{
					Err: err,
				}
				return
			}
			for _, objectSt := range result.Uploads {
				// NOTE: getTotalMultipartSize can make listing incomplete uploads slower.
				objectSt.Size, err = a.getTotalMultipartSize(bucket, objectSt.Key, objectSt.UploadID)
				if err != nil {
					ch <- ObjectMultipartStat{
						Err: err,
					}
				}
				ch <- objectSt
				multipartMarker = result.NextKeyMarker
				uploadIDMarker = result.NextUploadIDMarker
			}
			if !result.IsTruncated {
				break
			}
		}
	default:
		var multipartMarker string
		var uploadIDMarker string
		for {
			result, err := a.listMultipartUploads(bucket, multipartMarker, uploadIDMarker, prefix, "/", 1000)
			if err != nil {
				ch <- ObjectMultipartStat{
					Err: err,
				}
				return
			}
			multipartMarker = result.NextKeyMarker
			uploadIDMarker = result.NextUploadIDMarker
			for _, objectSt := range result.Uploads {
				objectSt.Size, err = a.getTotalMultipartSize(bucket, objectSt.Key, objectSt.UploadID)
				if err != nil {
					ch <- ObjectMultipartStat{
						Err: err,
					}
				}
				ch <- objectSt
			}
			for _, prefix := range result.CommonPrefixes {
				object := ObjectMultipartStat{}
				object.Key = prefix.Prefix
				object.Size = 0
				ch <- object
			}
			if !result.IsTruncated {
				break
			}
		}
	}
}

// ListIncompleteUploads - List incompletely uploaded multipart objects.
//
// ListIncompleteUploads is a channel based API implemented to facilitate ease of usage of S3 API ListMultipartUploads()
// by automatically recursively traversing all multipart objects on a given bucket if specified.
//
// Your input paramters are just bucket, prefix and recursive.
// If you enable recursive as 'true' this function will return back all the multipart objects in a given bucket.
//
//   api := client.New(....)
//   recursive := true
//   for message := range api.ListIncompleteUploads("mytestbucket", "starthere", recursive) {
//       fmt.Println(message)
//   }
//
func (a API) ListIncompleteUploads(bucket, prefix string, recursive bool) <-chan ObjectMultipartStat {
	objectMultipartStatCh := make(chan ObjectMultipartStat, 1000)
	go a.listIncompleteUploadsInRoutine(bucket, prefix, recursive, objectMultipartStatCh)
	return objectMultipartStatCh
}

// listObjectsInRoutine is an internal goroutine function called for listing objects.
// This function feeds data into channel.
func (a API) listObjectsInRoutine(bucket, prefix string, recursive bool, ch chan<- ObjectStat) {
	defer close(ch)
	if err := invalidBucketError(bucket); err != nil {
		ch <- ObjectStat{
			Err: err,
		}
		return
	}
	switch {
	case recursive == true:
		var marker string
		for {
			result, err := a.listObjects(bucket, marker, prefix, "", 1000)
			if err != nil {
				ch <- ObjectStat{
					Err: err,
				}
				return
			}
			for _, object := range result.Contents {
				ch <- object
				marker = object.Key
			}
			if !result.IsTruncated {
				break
			}
		}
	default:
		var marker string
		for {
			result, err := a.listObjects(bucket, marker, prefix, "/", 1000)
			if err != nil {
				ch <- ObjectStat{
					Err: err,
				}
				return
			}
			marker = result.NextMarker
			for _, object := range result.Contents {
				ch <- object
			}
			for _, prefix := range result.CommonPrefixes {
				object := ObjectStat{}
				object.Key = prefix.Prefix
				object.Size = 0
				ch <- object
			}
			if !result.IsTruncated {
				break
			}
		}
	}
}

// ListObjects - (List Objects) - List some objects or all recursively.
//
// ListObjects is a channel based API implemented to facilitate ease of usage of S3 API ListObjects()
// by automatically recursively traversing all objects on a given bucket if specified.
//
// Your input paramters are just bucket, prefix and recursive.
// If you enable recursive as 'true' this function will return back all the objects in a given bucket.
//
//   api := client.New(....)
//   recursive := true
//   for message := range api.ListObjects("mytestbucket", "starthere", recursive) {
//       fmt.Println(message)
//   }
//
func (a API) ListObjects(bucket string, prefix string, recursive bool) <-chan ObjectStat {
	ch := make(chan ObjectStat, 1000)
	go a.listObjectsInRoutine(bucket, prefix, recursive, ch)
	return ch
}

// listBucketsInRoutine is an internal go routine function called for listing buckets
// This function feeds data into channel
func (a API) listBucketsInRoutine(ch chan<- BucketStat) {
	defer close(ch)
	listAllMyBucketListResults, err := a.listBuckets()
	if err != nil {
		ch <- BucketStat{
			Err: err,
		}
		return
	}
	for _, bucket := range listAllMyBucketListResults.Buckets.Bucket {
		ch <- bucket
	}
}

// ListBuckets list of all buckets owned by the authenticated sender of the request.
//
// This call requires explicit authentication, no anonymous requests are allowed for listing buckets.
//
//   api := client.New(....)
//   for message := range api.ListBuckets() {
//       fmt.Println(message)
//   }
//
func (a API) ListBuckets() <-chan BucketStat {
	ch := make(chan BucketStat, 100)
	go a.listBucketsInRoutine(ch)
	return ch
}

func (a API) removeIncompleteUploadInRoutine(bucket, object string, errorCh chan<- error) {
	defer close(errorCh)
	if err := invalidBucketError(bucket); err != nil {
		errorCh <- err
		return
	}
	if err := invalidObjectError(object); err != nil {
		errorCh <- err
		return
	}
	listMultipartUplResult, err := a.listMultipartUploads(bucket, "", "", object, "", 1000)
	if err != nil {
		errorCh <- err
		return
	}
	for _, multiPartUpload := range listMultipartUplResult.Uploads {
		if object == multiPartUpload.Key {
			err := a.abortMultipartUpload(bucket, multiPartUpload.Key, multiPartUpload.UploadID)
			if err != nil {
				errorCh <- err
				return
			}
			return
		}
	}
	for {
		if !listMultipartUplResult.IsTruncated {
			break
		}
		listMultipartUplResult, err = a.listMultipartUploads(bucket,
			listMultipartUplResult.NextKeyMarker, listMultipartUplResult.NextUploadIDMarker, object, "", 1000)
		if err != nil {
			errorCh <- err
			return
		}
		for _, multiPartUpload := range listMultipartUplResult.Uploads {
			if object == multiPartUpload.Key {
				err := a.abortMultipartUpload(bucket, multiPartUpload.Key, multiPartUpload.UploadID)
				if err != nil {
					errorCh <- err
					return
				}
				return
			}
		}

	}
}

// RemoveIncompleteUpload - abort a specific in progress active multipart upload.
// Requires explicit authentication, no anonymous requests are allowed for multipart API.
func (a API) RemoveIncompleteUpload(bucket, object string) <-chan error {
	errorCh := make(chan error)
	go a.removeIncompleteUploadInRoutine(bucket, object, errorCh)
	return errorCh
}
