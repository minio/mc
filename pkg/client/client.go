/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"time"
	"unicode/utf8"
)

// Client - client interface
type Client interface {
	MultipartUpload

	// Common operations
	Stat() error
	List() (items []*Item, err error)
	//	ListNew() error

	// Bucket operations
	PutBucket() error

	// Object operations
	Get() (body io.ReadCloser, size int64, md5 string, err error)
	GetPartial(offset, length int64) (body io.ReadCloser, size int64, md5 string, err error)
	Put(md5 string, size int64) (io.WriteCloser, error)
	GetObjectMetadata() (item *Item, err error)
}

// MultipartUpload - multi part upload interface
type MultipartUpload interface {
	InitiateMultiPartUpload() (objectID string, err error)
	UploadPart(uploadID string, partNumber int) (md5hex string, err error)
	CompleteMultiPartUpload(uploadID string) (location, md5hex string, err error)
	AbortMultiPartUpload(uploadID string) error
	ListParts(uploadID string) (items *PartItems, err error)
}

// Part - part xml response
type Part struct {
	PartNumber   int
	LastModified time.Time
	ETag         string
	Size         int64
}

// PartItems - part xml items response
type PartItems struct {
	Key         string
	UploadID    string
	IsTruncated bool
	Part        []*Part
}

// Item - object item list
type Item struct {
	Name string
	Time time.Time
	Size int64
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
	match, _ := regexp.MatchString("^[a-z][a-z0-9\\-]+[a-z0-9]$", bucket)
	return match
}

// BySize implements sort.Interface for []Item based on the Size field.
type BySize []*Item

// Len -
func (a BySize) Len() int { return len(a) }

// Swap -
func (a BySize) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less -
func (a BySize) Less(i, j int) bool { return a[i].Size < a[j].Size }

// IsValidObject - verify object name in accordance with
//   - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
func IsValidObject(object string) bool {
	if len(object) > 1024 || len(object) == 0 {
		return false
	}
	if !utf8.ValidString(object) {
		return false
	}
	return true
}
