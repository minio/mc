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

package client

import (
	"io"
	"os"
	"time"
)

// Client - client interface
type Client interface {
	//	MultipartUpload - TODO

	// Common operations
	Stat() (content *Content, err error)
	List() <-chan ContentOnChannel
	ListRecursive() <-chan ContentOnChannel

	// Bucket operations
	PutBucket() error
	PutBucketACL(acl string) error

	// Object operations
	Get() (body io.ReadCloser, size int64, md5 string, err error)
	GetPartial(offset, length int64) (body io.ReadCloser, size int64, md5 string, err error)
	Put(md5 string, size int64) (io.WriteCloser, error)
}

// MultipartUpload - multi part upload interface
//type MultipartUpload interface {
//	InitiateMultiPartUpload() (objectID string, err error)
//	UploadPart(uploadID string, body io.ReadSeeker, contentLength, partNumber int64) (md5hex string, err error)
//	CompleteMultiPartUpload(uploadID string) (location, md5hex string, err error)
//	AbortMultiPartUpload(uploadID string) error
//	ListParts(uploadID string) (contents *PartContents, err error)
//}

// Part - part xml response
type Part struct {
	ETag         string
	LastModified time.Time
	PartNumber   int64
	Size         int64
}

// PartContents - part xml contents response
type PartContents struct {
	Key         string
	UploadID    string
	IsTruncated bool
	Parts       []*Part
}

// ContentOnChannel - List contents on channel
type ContentOnChannel struct {
	Content *Content
	Err     error
}

// Content - object content list
type Content struct {
	Name     string
	Time     time.Time
	Size     int64
	FileType os.FileMode
}
