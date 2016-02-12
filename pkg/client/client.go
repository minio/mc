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
	List(recursive, incomplete bool) <-chan *Content

	// Bucket operations
	MakeBucket(region string) *probe.Error
	GetBucketAccess() (access string, error *probe.Error)
	SetBucketAccess(access string) *probe.Error

	// I/O operations
	Get() (reader io.Reader, err *probe.Error)
	Put(reader io.Reader, size int64, contentType string, progress io.Reader) (n int64, err *probe.Error)

	// I/O operations with expiration
	ShareDownload(expires time.Duration) (string, *probe.Error)
	ShareUpload(bool, time.Duration, string) (map[string]string, *probe.Error)

	// Delete operations
	Remove(incomplete bool) *probe.Error

	// GetURL returns back internal url
	GetURL() URL
}

// Content container for content metadata
type Content struct {
	URL  URL
	Time time.Time
	Size int64
	Type os.FileMode
	Err  *probe.Error
}

// Config - see http://docs.amazonwebservices.com/AmazonS3/latest/dev/index.html?RESTAuthentication.html
type Config struct {
	AccessKey   string
	SecretKey   string
	Signature   string
	HostURL     string
	AppName     string
	AppVersion  string
	AppComments []string
	Debug       bool
}
