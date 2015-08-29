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

	"github.com/minio/minio/pkg/probe"
)

// Client - client interface
type Client interface {
	// Common operations
	Stat() (content *Content, err *probe.Error)
	List(recursive bool) <-chan ContentOnChannel

	// Bucket operations
	MakeBucket() *probe.Error
	SetBucketACL(acl string) *probe.Error

	// Object operations
	Share(expires time.Duration) (string, *probe.Error)
	GetObject(offset, length int64) (body io.ReadCloser, size int64, err *probe.Error)
	PutObject(size int64, data io.Reader) *probe.Error

	// URL returns back internal url
	URL() *URL
}

// ContentOnChannel - List contents on channel
type ContentOnChannel struct {
	Content *Content
	Err     *probe.Error
}

// Content container for content metadata
type Content struct {
	Name string
	Time time.Time
	Size int64
	Type os.FileMode
}
