/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
