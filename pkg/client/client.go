/*
 * Modern Copy, (C) 2015 Minio, Inc.
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

package client

import (
	"io"
	"regexp"
	"strconv"
	"time"
)

// Client - client interface
type Client interface {
	// Bucket operations
	PutBucket(bucket string) error
	StatBucket(bucket string) error
	ListBuckets() ([]*Bucket, error)
	ListObjects(bucket, keyPrefix string) (items []*Item, err error)

	// Object operations
	Get(bucket, object string) (body io.ReadCloser, size int64, md5 string, err error)
	GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, md5 string, err error)
	Put(bucket, object, md5 string, size int64, body io.Reader) error
	StatObject(bucket, object string) (size int64, date time.Time, err error)
}

// Bucket - carries s3 bucket reply header
type Bucket struct {
	Name         string
	CreationDate time.Time // 2006-02-03T16:45:09.000Z
}

// Item - object item list
type Item struct {
	Key          string
	LastModified time.Time
	Size         int64
}

// Prefix - common prefix
type Prefix struct {
	Prefix string
}

// IsValidBucketName reports whether bucket is a valid bucket name, per Amazon's naming restrictions.
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
func IsValidBucketName(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if bucket[0] == '.' || bucket[len(bucket)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucket); match == true {
		return false
	}
	// We don't support buckets with '.' in them
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", bucket)
	return match
}

/// Collection of standard errors

// APINotImplemented - api not implemented
type APINotImplemented struct {
	API string
}

func (e APINotImplemented) Error() string {
	return "API not implemented: " + e.API
}

// InvalidArgument - bad arguments provided
type InvalidArgument struct{}

func (e InvalidArgument) Error() string {
	return "invalid arguments"
}

// InvalidMaxKeys - invalid maxkeys provided
type InvalidMaxKeys struct {
	MaxKeys int
}

func (e InvalidMaxKeys) Error() string {
	return "invalid maxkeys: " + strconv.Itoa(e.MaxKeys)
}

// InvalidAuthorizationKey - invalid authorization key
type InvalidAuthorizationKey struct{}

func (e InvalidAuthorizationKey) Error() string {
	return "invalid authorization key"
}

// AuthorizationKeyEmpty - empty auth key provided
type AuthorizationKeyEmpty struct{}

func (e AuthorizationKeyEmpty) Error() string {
	return "authorization key empty"
}

// InvalidRange - invalid range requested
type InvalidRange struct {
	Offset int64
}

func (e InvalidRange) Error() string {
	return "invalid range offset: " + strconv.FormatInt(e.Offset, 10)
}

// GenericBucketError - generic bucket operations error
type GenericBucketError struct {
	Bucket string
}

// BucketNotFound - bucket requested does not exist
type BucketNotFound GenericBucketError

func (e BucketNotFound) Error() string {
	return "bucket " + e.Bucket + " not found"
}

// BucketExists - bucket exists
type BucketExists GenericBucketError

func (e BucketExists) Error() string {
	return "bucket " + e.Bucket + " exists"
}

// InvalidBucketName - bucket name invalid
type InvalidBucketName GenericBucketError

func (e InvalidBucketName) Error() string {
	return "Invalid bucketname " + e.Bucket
}

// GenericObjectError - generic object operations error
type GenericObjectError struct {
	Bucket string
	Object string
}

// ObjectNotFound - object requested does not exist
type ObjectNotFound GenericObjectError

func (e ObjectNotFound) Error() string {
	return "object " + e.Object + " not found in bucket " + e.Bucket
}

// InvalidObjectName - object requested is invalid
type InvalidObjectName GenericObjectError

func (e InvalidObjectName) Error() string {
	return "object " + e.Object + "at" + e.Bucket + "is invalid"
}

// ObjectExists - object exists
type ObjectExists GenericObjectError

func (e ObjectExists) Error() string {
	return "object " + e.Object + " exists"
}
