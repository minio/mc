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
	// Common operations
	Stat() (content *Content, err error)
	List() <-chan ContentOnChannel
	ListRecursive() <-chan ContentOnChannel

	// Bucket operations
	CreateBucket() error
	SetBucketACL(acl string) error

	// Object operations
	GetObject(offset, length uint64) (body io.ReadCloser, size uint64, md5 string, err error)
	CreateObject(md5 string, size uint64, data io.Reader) error
}

// ContentOnChannel - List contents on channel
type ContentOnChannel struct {
	Content *Content
	Err     error
}

// Content - object content list
type Content struct {
	Name string
	Time time.Time
	Size int64
	Type os.FileMode
}
