/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package objectstorage

import (
	"errors"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"
)

// API - object storage API interface
type API interface {
	// Bucket Read/Write/Stat operations
	BucketAPI

	// Object Read/Write/Stat operations
	ObjectAPI
}

// BucketAPI - bucket specific Read/Write/Stat interface
type BucketAPI interface {
	CreateBucket(bucket, acl, location string) error
	SetBucketACL(bucket, acl string) error
	StatBucket(bucket string) error
	DeleteBucket(bucket string) error

	ListObjects(bucket, prefix string, recursive bool) <-chan ObjectOnChannel
	ListBuckets() <-chan BucketOnChannel
}

// ObjectAPI - object specific Read/Write/Stat interface
type ObjectAPI interface {
	GetObject(bucket, object string, offset, length uint64) (io.ReadCloser, *ObjectMetadata, error)
	CreateObject(bucket, object string, size uint64, data io.Reader) (string, error)
	StatObject(bucket, object string) (*ObjectMetadata, error)
	DeleteObject(bucket, object string) error
}

// BucketOnChannel - bucket metadata over read channel
type BucketOnChannel struct {
	Data *BucketMetadata
	Err  error
}

// ObjectOnChannel - object metadata over read channel
type ObjectOnChannel struct {
	Data *ObjectMetadata
	Err  error
}

// BucketMetadata container for bucket metadata
type BucketMetadata struct {
	// The name of the bucket.
	Name string
	// Date the bucket was created.
	CreationDate time.Time
}

// ObjectMetadata container for object metadata
type ObjectMetadata struct {
	ETag         string
	Key          string
	LastModified time.Time
	Size         int64

	Owner struct {
		DisplayName string
		ID          string
	}

	// The class of storage used to store the object.
	StorageClass string
}

// Regions s3 region map used by bucket location constraint
var Regions = map[string]string{
	"us-gov-west-1":  "https://s3-fips-us-gov-west-1.amazonaws.com",
	"us-east-1":      "https://s3.amazonaws.com",
	"us-west-1":      "https://s3-us-west-1.amazonaws.com",
	"us-west-2":      "https://s3-us-west-2.amazonaws.com",
	"eu-west-1":      "https://s3-eu-west-1.amazonaws.com",
	"eu-central-1":   "https://s3-eu-central-1.amazonaws.com",
	"ap-southeast-1": "https://s3-ap-southeast-1.amazonaws.com",
	"ap-southeast-2": "https://s3-ap-southeast-2.amazonaws.com",
	"ap-northeast-1": "https://s3-ap-northeast-1.amazonaws.com",
	"sa-east-1":      "https://s3-sa-east-1.amazonaws.com",
	"cn-north-1":     "https://s3.cn-north-1.amazonaws.com.cn",
}

// getEndpoint fetches an endpoint based on region through the S3 Regions map
func getEndpoint(region string) string {
	return Regions[region]
}

type api struct {
	*lowLevelAPI
}

// Config - main configuration struct used by all to set endpoint, credentials, and other options for requests.
type Config struct {
	// Standard options
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Endpoint        string

	// Advanced options
	AcceptType string            // specify this to get server response in non XML style if server supports it
	UserAgent  string            // user override useful when objectstorage-go is used with in your application
	Transport  http.RoundTripper // custom transport usually for debugging, by default its nil
}

// MustGetEndpoint makes sure that a valid endpoint is provided all the time, even with false regions it will fall
// back to default, for no regions specified it chooses to default to "milkyway" and use endpoint as is
func (c *Config) MustGetEndpoint() string {
	switch {
	case strings.TrimSpace(c.Endpoint) != "" && strings.TrimSpace(c.Region) == "":
		if strings.Contains(strings.TrimSpace(c.Endpoint), "s3.amazonaws.com") {
			c.Region = "us-east-1"
			return getEndpoint(c.Region)
		}
		// for custom domains, there are no regions default to 'milkyway'
		c.Region = "milkyway"
		return c.Endpoint
	// if valid region provided override user provided endpoint
	case strings.TrimSpace(c.Region) != "":
		if endpoint := getEndpoint(strings.TrimSpace(c.Region)); endpoint != "" {
			c.Endpoint = endpoint
			return c.Endpoint
		}
		// fall back to custom Endpoint, if no valid region can be found
		return c.Endpoint
	}
	// if not endpoint or region sepcified default to us-east-1
	c.Region = "us-east-1"
	return getEndpoint(c.Region)
}

// Global constants
const (
	LibraryName    = "objectstorage-go/"
	LibraryVersion = "0.1"
)

// New - instantiate a new minio api client
func New(config *Config) API {
	// if not UserAgent provided set it to default
	if strings.TrimSpace(config.UserAgent) == "" {
		config.UserAgent = LibraryName + " (" + LibraryVersion + "; " + runtime.GOOS + "; " + runtime.GOARCH + ")"
	}
	return &api{&lowLevelAPI{config}}
}

/// Object operations

// GetObject retrieve object
//
// Additionally it also takes range arguments to download the specified range bytes of an object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a *api) GetObject(bucket, object string, offset, length uint64) (io.ReadCloser, *ObjectMetadata, error) {
	// get the the object
	// NOTE : returned md5sum could be the md5sum of the partial object itself
	// not the whole object depending on if offset range was requested or not
	body, objectMetadata, err := a.getObject(bucket, object, offset, length)
	if err != nil {
		return nil, nil, err
	}
	return body, objectMetadata, nil
}

// completedParts is a wrapper to make parts sortable by their part number
// multi part completion requires list of multi parts to be sorted
type completedParts []*completePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

// DefaultPartSize - default size per object after which PutObject becomes multipart
// one can change this value during a library import
var DefaultPartSize uint64 = 1024 * 1024 * 5

func (a *api) newObjectUpload(bucket, object string, data io.Reader) (string, error) {
	initiateMultipartUploadResult, err := a.initiateMultipartUpload(bucket, object)
	if err != nil {
		return "", err
	}
	uploadID := initiateMultipartUploadResult.UploadID
	completeMultipartUpload := new(completeMultipartUpload)
	for part := range MultiPart(data, DefaultPartSize, nil) {
		if part.Err != nil {
			return "", part.Err
		}
		completePart, err := a.uploadPart(bucket, object, uploadID, part.Num, part.Len, part.Data)
		if err != nil {
			return "", a.abortMultipartUpload(bucket, object, uploadID)
		}
		completeMultipartUpload.Part = append(completeMultipartUpload.Part, completePart)
	}
	sort.Sort(completedParts(completeMultipartUpload.Part))
	completeMultipartUploadResult, err := a.completeMultipartUpload(bucket, object, uploadID, completeMultipartUpload)
	if err != nil {
		return "", a.abortMultipartUpload(bucket, object, uploadID)
	}
	return completeMultipartUploadResult.ETag, nil
}

func (a *api) continueObjectUpload(bucket, object, uploadID string, data io.Reader) (string, error) {
	listObjectPartsResult, err := a.listObjectParts(bucket, object, uploadID, 0, 1000)
	if err != nil {
		return "", err
	}
	var skipParts []int
	completeMultipartUpload := new(completeMultipartUpload)
	for _, uploadedPart := range listObjectPartsResult.Part {
		completedPart := new(completePart)
		completedPart.PartNumber = uploadedPart.PartNumber
		completedPart.ETag = uploadedPart.ETag
		completeMultipartUpload.Part = append(completeMultipartUpload.Part, completedPart)
		skipParts = append(skipParts, uploadedPart.PartNumber)
	}
	for part := range MultiPart(data, DefaultPartSize, skipParts) {
		if part.Err != nil {
			return "", part.Err
		}
		completedPart, err := a.uploadPart(bucket, object, uploadID, part.Num, part.Len, part.Data)
		if err != nil {
			return "", a.abortMultipartUpload(bucket, object, uploadID)
		}
		completeMultipartUpload.Part = append(completeMultipartUpload.Part, completedPart)
	}
	sort.Sort(completedParts(completeMultipartUpload.Part))
	completeMultipartUploadResult, err := a.completeMultipartUpload(bucket, object, uploadID, completeMultipartUpload)
	if err != nil {
		return "", a.abortMultipartUpload(bucket, object, uploadID)
	}
	return completeMultipartUploadResult.ETag, nil
}

// CreateObject create an object in a bucket
//
// You must have WRITE permissions on a bucket to create an object
//
// This version of CreateObject automatically does multipart for more than 5MB worth of data
func (a *api) CreateObject(bucket, object string, size uint64, data io.Reader) (string, error) {
	if strings.TrimSpace(object) == "" {
		return "", errors.New("object name cannot be empty")
	}
	switch {
	case size < DefaultPartSize:
		// Single Part use case, use PutObject directly
		for part := range MultiPart(data, DefaultPartSize, nil) {
			if part.Err != nil {
				return "", part.Err
			}
			metadata, err := a.putObject(bucket, object, part.Len, part.Data)
			if err != nil {
				return "", err
			}
			return metadata.ETag, nil
		}
	default:
		listMultipartUploadsResult, err := a.listMultipartUploads(bucket, "", "", object, "", 1000)
		if err != nil {
			return "", err
		}
		var inProgress bool
		var inProgressUploadID string
		for _, upload := range listMultipartUploadsResult.Upload {
			if object == upload.Key {
				inProgress = true
				inProgressUploadID = upload.UploadID
			}
		}
		if !inProgress {
			return a.newObjectUpload(bucket, object, data)
		}
		return a.continueObjectUpload(bucket, object, inProgressUploadID, data)
	}
	return "", errors.New("Unexpected control flow")
}

// StatObject verify if object exists and you have permission to access it
func (a *api) StatObject(bucket, object string) (*ObjectMetadata, error) {
	return a.headObject(bucket, object)
}

// DeleteObject remove the object from a bucket
func (a *api) DeleteObject(bucket, object string) error {
	return a.deleteObject(bucket, object)
}

/// Bucket operations

// CreateBucket create a new bucket
//
// optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and location set to US Standard if one wishes to set
// different ACLs and Location one can set them properly.
//
// ACL valid values
// ------------------
// private - owner gets full access [DEFAULT]
// public-read - owner gets full access, others get read access
// public-read-write - owner gets full access, others get full access too
// ------------------
//
// Location valid values
// ------------------
// [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
// Default - US standard
func (a *api) CreateBucket(bucket, acl, location string) error {
	return a.putBucket(bucket, acl, location)
}

// SetBucketACL set the permissions on an existing bucket using access control lists (ACL)
//
// Currently supported are:
// ------------------
// private - owner gets full access
// public-read - owner gets full access, others get read access
// public-read-write - owner gets full access, others get full access too
// ------------------
func (a *api) SetBucketACL(bucket, acl string) error {
	return a.putBucketACL(bucket, acl)
}

// StatBucket verify if bucket exists and you have permission to access it
func (a *api) StatBucket(bucket string) error {
	return a.headBucket(bucket)
}

// DeleteBucket deletes the bucket named in the URI
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request
func (a *api) DeleteBucket(bucket string) error {
	return a.deleteBucket(bucket)
}

// listObjectsInRoutine is an internal goroutine function called for listing objects
// This function feeds data into channel
func (a *api) listObjectsInRoutine(bucket, prefix string, recursive bool, ch chan ObjectOnChannel) {
	defer close(ch)
	switch {
	case recursive == true:
		listBucketResult, err := a.listObjects(bucket, "", prefix, "", 1000)
		if err != nil {
			ch <- ObjectOnChannel{
				Data: nil,
				Err:  err,
			}
			return
		}
		for _, object := range listBucketResult.Contents {
			ch <- ObjectOnChannel{
				Data: object,
				Err:  nil,
			}
		}
		for {
			if !listBucketResult.IsTruncated {
				break
			}
			listBucketResult, err = a.listObjects(bucket, listBucketResult.Marker, prefix, "", 1000)
			if err != nil {
				ch <- ObjectOnChannel{
					Data: nil,
					Err:  err,
				}
				return
			}
			for _, object := range listBucketResult.Contents {
				ch <- ObjectOnChannel{
					Data: object,
					Err:  nil,
				}
				listBucketResult.Marker = object.Key
			}
		}
	default:
		listBucketResult, err := a.listObjects(bucket, "", prefix, "/", 1000)
		if err != nil {
			ch <- ObjectOnChannel{
				Data: nil,
				Err:  err,
			}
			return
		}
		for _, object := range listBucketResult.Contents {
			ch <- ObjectOnChannel{
				Data: object,
				Err:  nil,
			}
		}
		for _, prefix := range listBucketResult.CommonPrefixes {
			object := new(ObjectMetadata)
			object.Key = prefix.Prefix
			object.Size = 0
			ch <- ObjectOnChannel{
				Data: object,
				Err:  nil,
			}
		}
	}
}

// ListObjects - (List Objects) - List some objects or all recursively
//
// ListObjects is a channel based API implemented to facilitate ease of usage of S3 API ListObjects()
// by automatically recursively traversing all objects on a given bucket if specified.
//
// Your input paramters are just bucket, prefix and recursive
//
// If you enable recursive as 'true' this function will return back all the objects in a given bucket
//
//  eg:-
//         api := objectstorage.New(....)
//         for message := range api.ListObjects("mytestbucket", "starthere", true) {
//                 fmt.Println(message.Data)
//         }
//
func (a *api) ListObjects(bucket string, prefix string, recursive bool) <-chan ObjectOnChannel {
	ch := make(chan ObjectOnChannel)
	go a.listObjectsInRoutine(bucket, prefix, recursive, ch)
	return ch
}

// listBucketsInRoutine is an internal go routine function called for listing buckets
// This function feeds data into channel
func (a *api) listBucketsInRoutine(ch chan BucketOnChannel) {
	defer close(ch)
	listAllMyBucketListResults, err := a.listBuckets()
	if err != nil {
		ch <- BucketOnChannel{
			Data: nil,
			Err:  err,
		}
		return
	}
	for _, bucket := range listAllMyBucketListResults.Buckets.Bucket {
		ch <- BucketOnChannel{
			Data: bucket,
			Err:  nil,
		}
	}

}

// ListBuckets list of all buckets owned by the authenticated sender of the request
//
// NOTE:
//     This call requires explicit authentication, no anonymous
//     requests are allowed for listing buckets
//
//  eg:-
//         api := objectstorage.New(....)
//         for message := range api.ListBuckets() {
//                 fmt.Println(message.Data)
//         }
//
func (a *api) ListBuckets() <-chan BucketOnChannel {
	ch := make(chan BucketOnChannel)
	go a.listBucketsInRoutine(ch)
	return ch
}
