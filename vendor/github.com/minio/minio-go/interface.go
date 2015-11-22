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
	"io"
	"time"
)

// CloudStorageAPI - Cloud Storage API interface
type CloudStorageAPI interface {
	// Bucket Read/Write/Stat operations
	MakeBucket(bucket string, cannedACL BucketACL) error
	BucketExists(bucket string) error
	RemoveBucket(bucket string) error
	SetBucketACL(bucket string, cannedACL BucketACL) error
	GetBucketACL(bucket string) (BucketACL, error)

	ListBuckets() <-chan BucketStat
	ListObjects(bucket, prefix string, recursive bool) <-chan ObjectStat
	ListIncompleteUploads(bucket, prefix string, recursive bool) <-chan ObjectMultipartStat

	// Object Read/Write/Stat operations
	GetObject(bucket, object string) (io.ReadCloser, ObjectStat, error)
	GetPartialObject(bucket, object string, offset, length int64) (io.ReadCloser, ObjectStat, error)
	PutObject(bucket, object, contentType string, size int64, data io.Reader) error
	StatObject(bucket, object string) (ObjectStat, error)
	RemoveObject(bucket, object string) error
	RemoveIncompleteUpload(bucket, object string) <-chan error

	// Presigned operations
	PresignedGetObject(bucket, object string, expires time.Duration) (string, error)
	PresignedPutObject(bucket, object string, expires time.Duration) (string, error)
	PresignedPostPolicy(*PostPolicy) (map[string]string, error)
}
