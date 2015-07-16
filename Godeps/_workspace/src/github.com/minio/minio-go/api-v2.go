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

package minio

import (
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
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
	MakeBucket(bucket string, cannedACL BucketACL) error
	BucketExists(bucket string) error
	RemoveBucket(bucket string) error
	SetBucketACL(bucket string, cannedACL BucketACL) error
	GetBucketACL(bucket string) (BucketACL, error)

	ListBuckets() <-chan BucketStatCh
	ListObjects(bucket, prefix string, recursive bool) <-chan ObjectStatCh

	// Drop all incomplete uploads
	DropAllIncompleteUploads(bucket string) <-chan error
}

// ObjectAPI - object specific Read/Write/Stat interface
type ObjectAPI interface {
	GetObject(bucket, object string) (io.ReadCloser, ObjectStat, error)
	GetPartialObject(bucket, object string, offset, length int64) (io.ReadCloser, ObjectStat, error)
	PutObject(bucket, object, contentType string, size int64, data io.Reader) error
	StatObject(bucket, object string) (ObjectStat, error)
	RemoveObject(bucket, object string) error

	// Drop all incomplete uploads for a given object
	DropIncompleteUploads(bucket, object string) <-chan error
}

// BucketStatCh - bucket metadata over read channel
type BucketStatCh struct {
	Stat BucketStat
	Err  error
}

// ObjectStatCh - object metadata over read channel
type ObjectStatCh struct {
	Stat ObjectStat
	Err  error
}

// BucketStat container for bucket metadata
type BucketStat struct {
	// The name of the bucket.
	Name string
	// Date the bucket was created.
	CreationDate time.Time
}

// ObjectStat container for object metadata
type ObjectStat struct {
	ETag         string
	Key          string
	LastModified time.Time
	Size         int64
	ContentType  string

	Owner struct {
		DisplayName string
		ID          string
	}

	// The class of storage used to store the object.
	StorageClass string
}

// Regions s3 region map used by bucket location constraint
var regions = map[string]string{
	"s3-fips-us-gov-west-1.amazonaws.com": "us-gov-west-1",
	"s3.amazonaws.com":                    "us-east-1",
	"s3-external-1.amazonaws.com":         "us-east-1",
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

// getRegion returns a region based on its endpoint mapping.
func getRegion(endPoint string) (region string, err error) {
	u, err := url.Parse(endPoint)
	if err != nil {
		return "", err
	}

	if regions[u.Host] != "" {
		return regions[u.Host], nil
	}

	// Region cannot be empty according to Amazon S3 standard.
	// So we address all the four quadrants of our galaxy.
	return "milkyway", nil
}

// Config - main configuration struct used by all to set endpoint, credentials, and other options for requests.
type Config struct {
	// Standard options
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string

	// Advanced options
	// Specify this to get server response in non XML style if server supports it
	AcceptType string
	// Optional field. If empty, region is determined automatically.
	Region string

	// Expert options
	//
	// Set this to override default transport ``http.DefaultTransport``
	//
	// This transport is usually needed for debugging OR to add your own
	// custom TLS certificates on the client transport, for custom CA's and
	// certs which are not part of standard certificate authority
	//
	// For example :-
	//
	//  tr := &http.Transport{
	//          TLSClientConfig:    &tls.Config{RootCAs: pool},
	//          DisableCompression: true,
	//  }
	//
	Transport http.RoundTripper

	// internal
	// use SetUserAgent append to default, useful when minio-go is used with in your application
	userAgent      string
	isUserAgentSet bool // allow user agent's to be set only once
}

// Global constants
const (
	LibraryName    = "minio-go"
	LibraryVersion = "0.1.0"
)

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

type apiV2 struct {
	apiV1
}

// New - instantiate a new minio api client
func New(config Config) (API, error) {
	if config.Region == "" {
		region, err := getRegion(config.Endpoint)
		if err != nil {
			return apiV2{}, err
		}
		config.Region = region
	}
	config.SetUserAgent(LibraryName, LibraryVersion, runtime.GOOS, runtime.GOARCH)
	config.isUserAgentSet = false // default
	return apiV2{apiV1{&config}}, nil
}

/// Object operations

// GetObject retrieve object

// Downloads full object with no ranges, if you need ranges use GetPartialObject
func (a apiV2) GetObject(bucket, object string) (io.ReadCloser, ObjectStat, error) {
	if err := invalidBucketToError(bucket); err != nil {
		return nil, ObjectStat{}, err
	}
	if err := invalidObjectToError(object); err != nil {
		return nil, ObjectStat{}, err
	}
	// get object
	return a.getObject(bucket, object, 0, 0)
}

// GetPartialObject retrieve partial object
//
// Takes range arguments to download the specified range bytes of an object.
// Setting offset and length = 0 will download the full object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a apiV2) GetPartialObject(bucket, object string, offset, length int64) (io.ReadCloser, ObjectStat, error) {
	if err := invalidBucketToError(bucket); err != nil {
		return nil, ObjectStat{}, err
	}
	if err := invalidObjectToError(object); err != nil {
		return nil, ObjectStat{}, err
	}
	// get partial object
	return a.getObject(bucket, object, offset, length)
}

// completedParts is a wrapper to make parts sortable by their part number
// multi part completion requires list of multi parts to be sorted
type completedParts []completePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

// minimumPartSize minimum part size per object after which PutObject behaves internally as multipart
var minimumPartSize int64 = 1024 * 1024 * 5

// maxParts - unexported right now
var maxParts = int64(10000)

// maxPartSize - unexported right now
var maxPartSize int64 = 1024 * 1024 * 1024 * 5

// GetPartSize - calculate the optimal part size for the given objectSize
//
// NOTE: Assumption here is that for any given object upload to a S3 compatible object
// storage it will have the following parameters as constants
//
//  maxParts
//  maximumPartSize
//  minimumPartSize
//
// if a the partSize after division with maxParts is greater than minimumPartSize
// then choose that to be the new part size, if not return MinimumPartSize
//
// special case where it happens to be that partSize is indeed bigger than the
// maximum part size just return maxPartSize back
func getPartSize(objectSize int64) int64 {
	partSize := (objectSize / (maxParts - 1)) // make sure last part has enough buffer and handle this poperly
	{
		if partSize > minimumPartSize {
			if partSize > maxPartSize {
				return maxPartSize
			}
			return partSize
		}
		return minimumPartSize
	}
}

func (a apiV2) newObjectUpload(bucket, object, contentType string, size int64, data io.Reader) error {
	initiateMultipartUploadResult, err := a.initiateMultipartUpload(bucket, object)
	if err != nil {
		return err
	}
	uploadID := initiateMultipartUploadResult.UploadID
	completeMultipartUpload := completeMultipartUpload{}
	var totalLength int64
	for part := range chopper(data, getPartSize(size), nil) {
		if part.Err != nil {
			return part.Err
		}
		if part.Len < minimumPartSize {
			if (size - totalLength) != part.Len {
				return ErrorResponse{
					Code:    "IncompleteBody",
					Message: "IncompleteBody",
				}
			}
		}
		completePart, err := a.uploadPart(bucket, object, uploadID, part.Md5Sum, part.Num, part.Len, part.ReadSeeker)
		if err != nil {
			return err
		}
		totalLength += part.Len
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, completePart)
	}
	sort.Sort(completedParts(completeMultipartUpload.Parts))
	_, err = a.completeMultipartUpload(bucket, object, uploadID, completeMultipartUpload)
	if err != nil {
		return err
	}
	return nil
}

type partCh struct {
	Metadata partMetadata
	Err      error
}

func (a apiV2) listObjectPartsRecursive(bucket, object, uploadID string) <-chan partCh {
	partCh := make(chan partCh)
	go a.listObjectPartsRecursiveInRoutine(bucket, object, uploadID, partCh)
	return partCh
}

func (a apiV2) listObjectPartsRecursiveInRoutine(bucket, object, uploadID string, ch chan partCh) {
	defer close(ch)
	listObjectPartsResult, err := a.listObjectParts(bucket, object, uploadID, 0, 1000)
	if err != nil {
		ch <- partCh{
			Metadata: partMetadata{},
			Err:      err,
		}
		return
	}
	for _, uploadedPart := range listObjectPartsResult.Parts {
		ch <- partCh{
			Metadata: uploadedPart,
			Err:      nil,
		}
	}
	for {
		if !listObjectPartsResult.IsTruncated {
			break
		}
		listObjectPartsResult, err = a.listObjectParts(bucket, object, uploadID, listObjectPartsResult.NextPartNumberMarker, 1000)
		if err != nil {
			ch <- partCh{
				Metadata: partMetadata{},
				Err:      err,
			}
			return
		}
		for _, uploadedPart := range listObjectPartsResult.Parts {
			ch <- partCh{
				Metadata: uploadedPart,
				Err:      nil,
			}
		}
	}
}

func (a apiV2) continueObjectUpload(bucket, object, uploadID string, size int64, data io.Reader) error {
	var skipParts []skipPart
	completeMultipartUpload := completeMultipartUpload{}
	var totalLength int64
	for part := range a.listObjectPartsRecursive(bucket, object, uploadID) {
		if part.Err != nil {
			return part.Err
		}
		var completedPart completePart
		completedPart.PartNumber = part.Metadata.PartNumber
		completedPart.ETag = part.Metadata.ETag
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, completedPart)
		md5SumBytes, err := hex.DecodeString(strings.Trim(part.Metadata.ETag, "\"")) // trim off the odd double quotes
		if err != nil {
			return err
		}
		totalLength += part.Metadata.Size
		skipParts = append(skipParts, skipPart{
			md5sum:     md5SumBytes,
			partNumber: part.Metadata.PartNumber,
		})
	}
	for part := range chopper(data, getPartSize(size), skipParts) {
		if part.Err != nil {
			return part.Err
		}
		if part.Len < minimumPartSize {
			if (size - totalLength) != part.Len {
				return ErrorResponse{
					Code:    "IncompleteBody",
					Message: "IncompleteBody",
				}
			}
		}
		completedPart, err := a.uploadPart(bucket, object, uploadID, part.Md5Sum, part.Num, part.Len, part.ReadSeeker)
		if err != nil {
			return err
		}
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, completedPart)
	}
	sort.Sort(completedParts(completeMultipartUpload.Parts))
	_, err := a.completeMultipartUpload(bucket, object, uploadID, completeMultipartUpload)
	if err != nil {
		return err
	}
	return nil
}

type multiPartUploadCh struct {
	Metadata multiPartUpload
	Err      error
}

func (a apiV2) listMultipartUploadsRecursive(bucket, object string) <-chan multiPartUploadCh {
	ch := make(chan multiPartUploadCh)
	go a.listMultipartUploadsRecursiveInRoutine(bucket, object, ch)
	return ch
}

func (a apiV2) listMultipartUploadsRecursiveInRoutine(bucket, object string, ch chan multiPartUploadCh) {
	defer close(ch)
	listMultipartUploadsResult, err := a.listMultipartUploads(bucket, "", "", object, "", 1000)
	if err != nil {
		ch <- multiPartUploadCh{
			Metadata: multiPartUpload{},
			Err:      err,
		}
		return
	}
	for _, multiPartUpload := range listMultipartUploadsResult.Uploads {
		ch <- multiPartUploadCh{
			Metadata: multiPartUpload,
			Err:      nil,
		}
	}
	for {
		if !listMultipartUploadsResult.IsTruncated {
			break
		}
		listMultipartUploadsResult, err = a.listMultipartUploads(bucket,
			listMultipartUploadsResult.NextKeyMarker, listMultipartUploadsResult.NextUploadIDMarker, object, "", 1000)
		if err != nil {
			ch <- multiPartUploadCh{
				Metadata: multiPartUpload{},
				Err:      err,
			}
			return
		}
		for _, multiPartUpload := range listMultipartUploadsResult.Uploads {
			ch <- multiPartUploadCh{
				Metadata: multiPartUpload,
				Err:      nil,
			}
		}
	}
}

// PutObject create an object in a bucket
//
// You must have WRITE permissions on a bucket to create an object
//
// This version of PutObject automatically does multipart for more than 5MB worth of data
func (a apiV2) PutObject(bucket, object, contentType string, size int64, data io.Reader) error {
	if err := invalidBucketToError(bucket); err != nil {
		return err
	}
	if err := invalidArgumentToError(object); err != nil {
		return err
	}
	switch {
	case size < minimumPartSize:
		// Single Part use case, use PutObject directly
		for part := range chopper(data, minimumPartSize, nil) {
			if part.Err != nil {
				return part.Err
			}
			if part.Len != size {
				return ErrorResponse{
					Code:    "IncompleteBody",
					Message: "IncompleteBody",
				}
			}
			_, err := a.putObject(bucket, object, contentType, part.Md5Sum, part.Len, part.ReadSeeker)
			if err != nil {
				return err
			}
			return nil
		}
	default:
		var inProgress bool
		var inProgressUploadID string
		for mpUpload := range a.listMultipartUploadsRecursive(bucket, object) {
			if mpUpload.Err != nil {
				return mpUpload.Err
			}
			if mpUpload.Metadata.Key == object {
				inProgress = true
				inProgressUploadID = mpUpload.Metadata.UploadID
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

// StatObject verify if object exists and you have permission to access it
func (a apiV2) StatObject(bucket, object string) (ObjectStat, error) {
	if err := invalidBucketToError(bucket); err != nil {
		return ObjectStat{}, err
	}
	if err := invalidObjectToError(object); err != nil {
		return ObjectStat{}, err
	}
	return a.headObject(bucket, object)
}

// RemoveObject remove the object from a bucket
func (a apiV2) RemoveObject(bucket, object string) error {
	if err := invalidBucketToError(bucket); err != nil {
		return err
	}
	if err := invalidObjectToError(object); err != nil {
		return err
	}
	return a.deleteObject(bucket, object)
}

/// Bucket operations

// MakeBucket make a new bucket
//
// optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and location set to US Standard if one wishes to set
// different ACLs and Location one can set them properly.
//
// ACL valid values
//
//  private - owner gets full access [default]
//  public-read - owner gets full access, all others get read access
//  public-read-write - owner gets full access, all others get full access too
//  authenticated-read - owner gets full access, authenticated users get read access
//
// Location valid values which are automatically derived from config endpoint
//
//  [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
//  Default - US standard
func (a apiV2) MakeBucket(bucket string, acl BucketACL) error {
	if err := invalidBucketToError(bucket); err != nil {
		return err
	}
	if !acl.isValidBucketACL() {
		return invalidArgumentToError("")
	}
	location, _ := getRegion(a.config.Endpoint)
	if location == "milkyway" {
		location = ""
	}
	if location == "us-east-1" {
		location = ""
	}
	return a.putBucket(bucket, string(acl), location)
}

// SetBucketACL set the permissions on an existing bucket using access control lists (ACL)
//
// For example
//
//  private - owner gets full access [default]
//  public-read - owner gets full access, all others get read access
//  public-read-write - owner gets full access, all others get full access too
//  authenticated-read - owner gets full access, authenticated users get read access
//
func (a apiV2) SetBucketACL(bucket string, acl BucketACL) error {
	if err := invalidBucketToError(bucket); err != nil {
		return err
	}
	if !acl.isValidBucketACL() {
		return invalidArgumentToError("")
	}
	return a.putBucketACL(bucket, string(acl))
}

// GetBucketACL get the permissions on an existing bucket
//
// Returned values are:
//
//  private - owner gets full access
//  public-read - owner gets full access, others get read access
//  public-read-write - owner gets full access, others get full access too
//  authenticated-read - owner gets full access, authenticated users get read access
//
func (a apiV2) GetBucketACL(bucket string) (BucketACL, error) {
	if err := invalidBucketToError(bucket); err != nil {
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

// BucketExists verify if bucket exists and you have permission to access it
func (a apiV2) BucketExists(bucket string) error {
	if err := invalidBucketToError(bucket); err != nil {
		return err
	}
	return a.headBucket(bucket)
}

// RemoveBucket deletes the bucket named in the URI
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request
func (a apiV2) RemoveBucket(bucket string) error {
	if err := invalidBucketToError(bucket); err != nil {
		return err
	}
	return a.deleteBucket(bucket)
}

// listObjectsInRoutine is an internal goroutine function called for listing objects
// This function feeds data into channel
func (a apiV2) listObjectsInRoutine(bucket, prefix string, recursive bool, ch chan ObjectStatCh) {
	defer close(ch)
	if err := invalidBucketToError(bucket); err != nil {
		ch <- ObjectStatCh{
			Stat: ObjectStat{},
			Err:  err,
		}
		return
	}
	switch {
	case recursive == true:
		var marker string
		for {
			result, err := a.listObjects(bucket, marker, prefix, "", 1000)
			if err != nil {
				ch <- ObjectStatCh{
					Stat: ObjectStat{},
					Err:  err,
				}
				return
			}
			for _, object := range result.Contents {
				ch <- ObjectStatCh{
					Stat: object,
					Err:  nil,
				}
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
				ch <- ObjectStatCh{
					Stat: ObjectStat{},
					Err:  err,
				}
				return
			}
			marker = result.NextMarker
			for _, object := range result.Contents {
				ch <- ObjectStatCh{
					Stat: object,
					Err:  nil,
				}
			}
			for _, prefix := range result.CommonPrefixes {
				object := ObjectStat{}
				object.Key = prefix.Prefix
				object.Size = 0
				ch <- ObjectStatCh{
					Stat: object,
					Err:  nil,
				}
			}
			if !result.IsTruncated {
				break
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
//         api := client.New(....)
//         for message := range api.ListObjects("mytestbucket", "starthere", true) {
//                 fmt.Println(message.Stat)
//         }
//
func (a apiV2) ListObjects(bucket string, prefix string, recursive bool) <-chan ObjectStatCh {
	ch := make(chan ObjectStatCh)
	go a.listObjectsInRoutine(bucket, prefix, recursive, ch)
	return ch
}

// listBucketsInRoutine is an internal go routine function called for listing buckets
// This function feeds data into channel
func (a apiV2) listBucketsInRoutine(ch chan BucketStatCh) {
	defer close(ch)
	listAllMyBucketListResults, err := a.listBuckets()
	if err != nil {
		ch <- BucketStatCh{
			Stat: BucketStat{},
			Err:  err,
		}
		return
	}
	for _, bucket := range listAllMyBucketListResults.Buckets.Bucket {
		ch <- BucketStatCh{
			Stat: bucket,
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
//         api := client.New(....)
//         for message := range api.ListBuckets() {
//                 fmt.Println(message.Stat)
//         }
//
func (a apiV2) ListBuckets() <-chan BucketStatCh {
	ch := make(chan BucketStatCh)
	go a.listBucketsInRoutine(ch)
	return ch
}

func (a apiV2) dropIncompleteUploadsInRoutine(bucket, object string, errorCh chan error) {
	defer close(errorCh)
	if err := invalidBucketToError(bucket); err != nil {
		errorCh <- err
		return
	}
	if err := invalidObjectToError(object); err != nil {
		errorCh <- err
		return
	}
	listMultipartUploadsResult, err := a.listMultipartUploads(bucket, "", "", object, "", 1000)
	if err != nil {
		errorCh <- err
		return
	}
	for _, multiPartUpload := range listMultipartUploadsResult.Uploads {
		err := a.abortMultipartUpload(bucket, multiPartUpload.Key, multiPartUpload.UploadID)
		if err != nil {
			errorCh <- err
			return
		}
	}
	for {
		if !listMultipartUploadsResult.IsTruncated {
			break
		}
		listMultipartUploadsResult, err = a.listMultipartUploads(bucket,
			listMultipartUploadsResult.NextKeyMarker, listMultipartUploadsResult.NextUploadIDMarker, object, "", 1000)
		if err != nil {
			errorCh <- err
			return
		}
		for _, multiPartUpload := range listMultipartUploadsResult.Uploads {
			err := a.abortMultipartUpload(bucket, multiPartUpload.Key, multiPartUpload.UploadID)
			if err != nil {
				errorCh <- err
				return
			}
		}

	}
}

//
//
// NOTE:
//   These set of calls require explicit authentication, no anonymous
//   requests are allowed for multipart API

// DropIncompleteUploads - abort a specific in progress active multipart upload
func (a apiV2) DropIncompleteUploads(bucket, object string) <-chan error {
	errorCh := make(chan error)
	go a.dropIncompleteUploadsInRoutine(bucket, object, errorCh)
	return errorCh
}

func (a apiV2) dropAllIncompleteUploadsInRoutine(bucket string, errorCh chan error) {
	defer close(errorCh)
	if err := invalidBucketToError(bucket); err != nil {
		errorCh <- err
		return
	}
	listMultipartUploadsResult, err := a.listMultipartUploads(bucket, "", "", "", "", 1000)
	if err != nil {
		errorCh <- err
		return
	}
	for _, multiPartUpload := range listMultipartUploadsResult.Uploads {
		err := a.abortMultipartUpload(bucket, multiPartUpload.Key, multiPartUpload.UploadID)
		if err != nil {
			errorCh <- err
			return
		}
	}
	for {
		if !listMultipartUploadsResult.IsTruncated {
			break
		}
		listMultipartUploadsResult, err = a.listMultipartUploads(bucket,
			listMultipartUploadsResult.NextKeyMarker, listMultipartUploadsResult.NextUploadIDMarker, "", "", 1000)
		if err != nil {
			errorCh <- err
			return
		}
		for _, multiPartUpload := range listMultipartUploadsResult.Uploads {
			err := a.abortMultipartUpload(bucket, multiPartUpload.Key, multiPartUpload.UploadID)
			if err != nil {
				errorCh <- err
				return
			}
		}

	}
}

// DropAllIncompleteUploads - abort all inprogress active multipart uploads
func (a apiV2) DropAllIncompleteUploads(bucket string) <-chan error {
	errorCh := make(chan error)
	go a.dropAllIncompleteUploadsInRoutine(bucket, errorCh)
	return errorCh
}
