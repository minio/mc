/*
 * MinIO Client (C) 2015 MinIO, Inc.
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
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/encrypt"
	"github.com/minio/minio/pkg/bucket/object/tagging"
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
	Stat(isIncomplete, isFetchMeta, isPreserve bool, sse encrypt.ServerSide) (content *ClientContent, err *probe.Error)
	List(isRecursive, isIncomplete, isFetchMeta bool, showDir DirOpt) <-chan *ClientContent

	// Bucket operations
	MakeBucket(region string, ignoreExisting, withLock bool) *probe.Error
	SetObjectLockConfig(mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit) *probe.Error
	GetObjectLockConfig() (mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit, perr *probe.Error)

	// Access policy operations.
	GetAccess() (access string, policyJSON string, error *probe.Error)
	GetAccessRules() (policyRules map[string]string, error *probe.Error)
	SetAccess(access string, isJSON bool) *probe.Error

	// I/O operations
	Copy(source string, size int64, progress io.Reader, srcSSE, tgtSSE encrypt.ServerSide, metadata map[string]string, disableMultipart bool) *probe.Error

	// Runs select expression on object storage on specific files.
	Select(expression string, sse encrypt.ServerSide, opts SelectObjectOpts) (io.ReadCloser, *probe.Error)

	// I/O operations with metadata.
	Get(sse encrypt.ServerSide) (reader io.ReadCloser, err *probe.Error)
	Put(ctx context.Context, reader io.Reader, size int64, metadata map[string]string, progress io.Reader, sse encrypt.ServerSide, disableMultipart bool) (n int64, err *probe.Error)
	// Object Locking related API
	PutObjectRetention(mode *minio.RetentionMode, retainUntilDate *time.Time, bypassGovernance bool) *probe.Error
	PutObjectLegalHold(hold *minio.LegalHoldStatus) *probe.Error

	// I/O operations with expiration
	ShareDownload(expires time.Duration) (string, *probe.Error)
	ShareUpload(bool, time.Duration, string) (string, map[string]string, *probe.Error)

	// Watch events
	Watch(params watchParams) (*WatchObject, *probe.Error)

	// Delete operations
	Remove(isIncomplete, isRemoveBucket bool, contentCh <-chan *ClientContent) (errorCh <-chan *probe.Error)

	// GetURL returns back internal url
	GetURL() ClientURL

	AddUserAgent(app, version string)

	// Object Tag operations
	GetObjectTagging() (tagging.Tagging, *probe.Error)
	SetObjectTagging(tagMap map[string]string) *probe.Error
	DeleteObjectTagging() *probe.Error
}

// ClientContent - Content container for content metadata
type ClientContent struct {
	URL          ClientURL
	Time         time.Time
	Size         int64
	Type         os.FileMode
	StorageClass string
	Metadata     map[string]string
	UserMetadata map[string]string
	ETag         string
	Expires      time.Time
	Retention    bool
	Err          *probe.Error
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

// SelectObjectOpts - opts entered for select API
type SelectObjectOpts struct {
	InputSerOpts    map[string]map[string]string
	OutputSerOpts   map[string]map[string]string
	CompressionType minio.SelectCompressionType
}
