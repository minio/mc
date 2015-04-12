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
	Put(bucket, object, md5 string, size int64) (io.WriteCloser, error)
	StatObject(bucket, object string) (size int64, date time.Time, err error)
}

type WriteErrorCloser interface {
	io.WriteCloser

	CloseWithError(error) error
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

// BySize implements sort.Interface for []Item based on the Size field.
type BySize []*Item

// Len -
func (a BySize) Len() int { return len(a) }

// Swap -
func (a BySize) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less -
func (a BySize) Less(i, j int) bool { return a[i].Size < a[j].Size }
