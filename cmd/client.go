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

package cmd

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go"
)

// DirOpt - list directory option.
type DirOpt int8

const (
	// DirNone - do not include directories in the list.
	DirNone DirOpt = iota
	// DirFirst - include directories before objects in the list.
	DirFirst
	// DirLast - include directories after objects in the list.
	DirLast
)

// Default number of multipart workers for a Put operation.
const defaultMultipartThreadsNum = 4

// Client - client interface
type Client interface {
	// Common operations
	Stat(isIncomplete, isFetchMeta bool, sseKey string) (content *clientContent, err *probe.Error)
	List(isRecursive, isIncomplete bool, showDir DirOpt) <-chan *clientContent

	// Bucket operations
	MakeBucket(region string, ignoreExisting bool) *probe.Error

	// Access policy operations.
	GetAccess() (access string, error *probe.Error)
	GetAccessRules() (policyRules map[string]string, error *probe.Error)
	SetAccess(access string) *probe.Error

	// I/O operations
	Copy(source string, size int64, progress io.Reader, srcSSEKey, tgtSSEKey string) *probe.Error

	// Runs select expression on object storage on specific files.
	Select(expression string, sseKey string) (io.ReadCloser, *probe.Error)

	// I/O operations with metadata.
	Get(sseKey string) (reader io.Reader, err *probe.Error)
	Put(ctx context.Context, reader io.Reader, size int64, metadata map[string]string, progress io.Reader, sseKey string) (n int64, err *probe.Error)

	// I/O operations with expiration
	ShareDownload(expires time.Duration) (string, *probe.Error)
	ShareUpload(bool, time.Duration, string) (string, map[string]string, *probe.Error)

	// Watch events
	Watch(params watchParams) (*watchObject, *probe.Error)

	// Delete operations
	Remove(isIncomplete bool, contentCh <-chan *clientContent) (errorCh <-chan *probe.Error)

	// GetURL returns back internal url
	GetURL() clientURL
}

// Content container for content metadata
type clientContent struct {
	URL               clientURL
	Time              time.Time
	Size              int64
	Type              os.FileMode
	Metadata          map[string]string
	ETag              string
	EncryptionHeaders map[string]string
	Err               *probe.Error
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
	Insecure    bool
	Lookup      minio.BucketLookupType
}
