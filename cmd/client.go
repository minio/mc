// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"maps"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/klauspost/compress/gzhttp"
	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/mc/pkg/limiter"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/v3/env"
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

// GetOptions holds options of the GET operation
type GetOptions struct {
	SSE        encrypt.ServerSide
	VersionID  string
	Zip        bool
	RangeStart int64
	PartNumber int
	Preserve   bool
}

// PutOptions holds options for PUT operation
type PutOptions struct {
	metadata              map[string]string
	sse                   encrypt.ServerSide
	md5, disableMultipart bool
	isPreserve            bool
	storageClass          string
	multipartSize         uint64
	multipartThreads      uint
	concurrentStream      bool
	ifNotExists           bool
	checksum              minio.ChecksumType
}

// StatOptions holds options of the HEAD operation
type StatOptions struct {
	incomplete         bool
	preserve           bool
	sse                encrypt.ServerSide
	timeRef            time.Time
	versionID          string
	includeVersions    bool
	isZip              bool
	ignoreBucketExists bool
	headOnly           bool
}

// BucketStatOptions - bucket stat.
type BucketStatOptions struct {
	bucket             string
	ignoreBucketExists bool
}

// ListOptions holds options for listing operation
type ListOptions struct {
	Recursive         bool
	Incomplete        bool
	WithMetadata      bool
	WithOlderVersions bool
	WithDeleteMarkers bool
	ListZip           bool
	TimeRef           time.Time
	ShowDir           DirOpt
	Count             int
}

// CopyOptions holds options for copying operation
type CopyOptions struct {
	versionID        string
	size             int64
	srcSSE, tgtSSE   encrypt.ServerSide
	metadata         map[string]string
	disableMultipart bool
	isPreserve       bool
	storageClass     string
}

// Client - client interface
type Client interface {
	// Common operations
	Stat(ctx context.Context, opts StatOptions) (content *ClientContent, err *probe.Error)
	List(ctx context.Context, opts ListOptions) <-chan *ClientContent

	// Bucket operations
	MakeBucket(ctx context.Context, region string, ignoreExisting, withLock bool) *probe.Error
	RemoveBucket(ctx context.Context, forceRemove bool) *probe.Error
	ListBuckets(ctx context.Context) ([]*ClientContent, *probe.Error)

	// Object lock config
	SetObjectLockConfig(ctx context.Context, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit) *probe.Error
	GetObjectLockConfig(ctx context.Context) (status string, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, perr *probe.Error)

	// Access policy operations.
	GetAccess(ctx context.Context) (access, policyJSON string, err *probe.Error)
	GetAccessRules(ctx context.Context) (policyRules map[string]string, err *probe.Error)
	SetAccess(ctx context.Context, access string, isJSON bool) *probe.Error

	// I/O operations
	Copy(ctx context.Context, source string, opts CopyOptions, progress io.Reader) *probe.Error

	// Runs select expression on object storage on specific files.
	Select(ctx context.Context, expression string, sse encrypt.ServerSide, opts SelectObjectOpts) (io.ReadCloser, *probe.Error)

	// I/O operations with metadata.
	Get(ctx context.Context, opts GetOptions) (reader io.ReadCloser, content *ClientContent, err *probe.Error)
	Put(ctx context.Context, reader io.Reader, size int64, progress io.Reader, opts PutOptions) (n int64, err *probe.Error)

	// Object Locking related API
	PutObjectRetention(ctx context.Context, versionID string, mode minio.RetentionMode, retainUntilDate time.Time, bypassGovernance bool) *probe.Error
	GetObjectRetention(ctx context.Context, versionID string) (minio.RetentionMode, time.Time, *probe.Error)
	PutObjectLegalHold(ctx context.Context, versionID string, hold minio.LegalHoldStatus) *probe.Error
	GetObjectLegalHold(ctx context.Context, versionID string) (minio.LegalHoldStatus, *probe.Error)

	// I/O operations with expiration
	ShareDownload(ctx context.Context, versionID string, expires time.Duration) (string, *probe.Error)
	ShareUpload(context.Context, bool, time.Duration, string) (string, map[string]string, *probe.Error)

	// Watch events
	Watch(ctx context.Context, options WatchOptions) (*WatchObject, *probe.Error)

	// Delete operations
	Remove(ctx context.Context, isIncomplete, isRemoveBucket, isBypass, isForceDel bool, contentCh <-chan *ClientContent) (errorCh <-chan RemoveResult)
	// GetURL returns back internal url
	GetURL() ClientURL
	AddUserAgent(app, version string)

	// Tagging operations
	GetTags(ctx context.Context, versionID string) (map[string]string, *probe.Error)
	SetTags(ctx context.Context, versionID, tags string) *probe.Error
	DeleteTags(ctx context.Context, versionID string) *probe.Error

	// Lifecycle operations
	GetLifecycle(ctx context.Context) (*lifecycle.Configuration, time.Time, *probe.Error)
	SetLifecycle(ctx context.Context, config *lifecycle.Configuration) *probe.Error

	// Versioning operations
	GetVersion(ctx context.Context) (minio.BucketVersioningConfiguration, *probe.Error)
	SetVersion(ctx context.Context, status string, prefixes []string, excludeFolders bool) *probe.Error
	// Replication operations
	GetReplication(ctx context.Context) (replication.Config, *probe.Error)
	SetReplication(ctx context.Context, cfg *replication.Config, opts replication.Options) *probe.Error
	RemoveReplication(ctx context.Context) *probe.Error
	GetReplicationMetrics(ctx context.Context) (replication.MetricsV2, *probe.Error)
	ResetReplication(ctx context.Context, before time.Duration, arn string) (replication.ResyncTargetsInfo, *probe.Error)
	ReplicationResyncStatus(ctx context.Context, arn string) (rinfo replication.ResyncTargetsInfo, err *probe.Error)

	// Encryption operations
	GetEncryption(ctx context.Context) (string, string, *probe.Error)
	SetEncryption(ctx context.Context, algorithm, kmsKeyID string) *probe.Error
	DeleteEncryption(ctx context.Context) *probe.Error
	// Bucket info operation
	GetBucketInfo(ctx context.Context) (BucketInfo, *probe.Error)

	// Restore an object
	Restore(ctx context.Context, versionID string, days int) *probe.Error

	// OD operations
	GetPart(ctx context.Context, part int) (io.ReadCloser, *probe.Error)
	PutPart(ctx context.Context, reader io.Reader, size int64, progress io.Reader, opts PutOptions) (n int64, err *probe.Error)

	// Cors operations
	GetBucketCors(ctx context.Context) (*cors.Config, *probe.Error)
	SetBucketCors(ctx context.Context, corsXML []byte) *probe.Error
	DeleteBucketCors(ctx context.Context) *probe.Error
}

// ClientContent - Content container for content metadata
type ClientContent struct {
	URL          ClientURL
	BucketName   string // only valid and set for client-type objectStorage
	Time         time.Time
	Size         int64
	Type         os.FileMode
	StorageClass string
	Metadata     map[string]string
	Tags         map[string]string
	UserMetadata map[string]string
	Checksum     map[string]string
	ETag         string
	Expires      time.Time

	Expiration       time.Time
	ExpirationRuleID string

	RetentionEnabled  bool
	RetentionMode     string
	RetentionDuration string
	BypassGovernance  bool
	LegalHoldEnabled  bool
	LegalHold         string
	VersionID         string
	IsDeleteMarker    bool
	IsLatest          bool
	ReplicationStatus string

	Restore *minio.RestoreInfo

	Err *probe.Error
}

// Config - see http://docs.amazonwebservices.com/AmazonS3/latest/dev/index.html?RESTAuthentication.html
type Config struct {
	Alias             string
	AccessKey         string
	SecretKey         string
	SessionToken      string
	Signature         string
	HostURL           string
	AppName           string
	AppVersion        string
	Debug             bool
	Insecure          bool
	Lookup            minio.BucketLookupType
	ConnReadDeadline  time.Duration
	ConnWriteDeadline time.Duration
	UploadLimit       int64
	DownloadLimit     int64
	Transport         http.RoundTripper
}

// getCredsChain returns an []credentials.Provider array for the config
// and the STS configuration (if present)
func (config *Config) getCredsChain() ([]credentials.Provider, *probe.Error) {
	var credsChain []credentials.Provider
	// if an STS endpoint is set, we will add that to the chain
	if stsEndpoint := env.Get("MC_STS_ENDPOINT_"+config.Alias, ""); stsEndpoint != "" {
		// set AWS_WEB_IDENTITY_TOKEN_FILE is MC_WEB_IDENTITY_TOKEN_FILE is set
		if val := env.Get("MC_WEB_IDENTITY_TOKEN_FILE_"+config.Alias, ""); val != "" {
			os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", val)
			if val := env.Get("MC_ROLE_ARN_"+config.Alias, ""); val != "" {
				os.Setenv("AWS_ROLE_ARN", val)
			}
			if val := env.Get("MC_ROLE_SESSION_NAME_"+config.Alias, randString(32, rand.NewSource(time.Now().UnixNano()), "mc-session-name-")); val != "" {
				os.Setenv("AWS_ROLE_SESSION_NAME", val)
			}
		}
		stsEndpointURL, err := url.Parse(stsEndpoint)
		if err != nil {
			return nil, probe.NewError(fmt.Errorf("Error parsing sts endpoint: %w", err))
		}
		credsSts := &credentials.IAM{
			Client: &http.Client{
				Transport: config.getTransport(),
			},
			Endpoint: stsEndpointURL.String(),
		}
		credsChain = append(credsChain, credsSts)
	}

	signType := credentials.SignatureV4
	if strings.EqualFold(config.Signature, "s3v2") {
		signType = credentials.SignatureV2
	}

	// Credentials
	creds := &credentials.Static{
		Value: credentials.Value{
			AccessKeyID:     config.AccessKey,
			SecretAccessKey: config.SecretKey,
			SessionToken:    config.SessionToken,
			SignerType:      signType,
		},
	}
	credsChain = append(credsChain, creds)
	return credsChain, nil
}

// getTransport returns a corresponding *http.Transport for the *Config
// set withS3v2 bool to true to add traceV2 tracer.
func (config *Config) getTransport() http.RoundTripper {
	if config.Transport == nil {
		config.initTransport(true)
	}
	return config.Transport
}

func (config *Config) isTLS() bool {
	if stsEndpoint := env.Get("MC_STS_ENDPOINT_"+config.Alias, ""); stsEndpoint != "" {
		stsEndpointURL, err := url.Parse(stsEndpoint)
		if err != nil {
			return false
		}
		return isHostTLS(config) || stsEndpointURL.Scheme == "https"
	}
	return isHostTLS(config)
}

func (config *Config) initTransport(withS3v2 bool) {
	var transport http.RoundTripper

	useTLS := config.isTLS()

	if config.Transport != nil {
		transport = config.Transport
	} else {
		tr := &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           newCustomDialContext(config),
			MaxIdleConnsPerHost:   1024,
			WriteBufferSize:       32 << 10, // 32KiB moving up from 4KiB default
			ReadBufferSize:        32 << 10, // 32KiB moving up from 4KiB default
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
			// Set this value so that the underlying transport round-tripper
			// doesn't try to auto decode the body of objects with
			// content-encoding set to `gzip`.
			//
			// Refer:
			//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
			DisableCompression: true,
		}
		if useTLS {
			tr.DialTLSContext = newCustomDialTLSContext(&tls.Config{
				RootCAs:            globalRootCAs,
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: config.Insecure,
			})

			// Because we create a custom TLSClientConfig, we have to opt-in to HTTP/2.
			// See https://github.com/golang/go/issues/14275
			//
			// TODO: Enable http2.0 when upstream issues related to HTTP/2 are fixed.
			//
			// if e = http2.ConfigureTransport(tr); e != nil {
			// 	return nil, probe.NewError(e)
			// }
		}

		if len(globalCustomHeader) > 0 {
			transport = &headerTransport{
				RoundTripper: tr,
				customHeader: globalCustomHeader.Clone(),
			}
		} else {
			transport = tr
		}

	}

	transport = limiter.New(config.UploadLimit, config.DownloadLimit, transport)

	if config.Debug {
		if strings.EqualFold(config.Signature, "S3v4") {
			transport = httptracer.GetNewTraceTransport(newTraceV4(), transport)
		} else if strings.EqualFold(config.Signature, "S3v2") && withS3v2 {
			transport = httptracer.GetNewTraceTransport(newTraceV2(), transport)
		}
	} else {
		if !globalJSONLine && !globalJSON {
			transport = notifyExpiringTLS{transport: transport}
		}
	}

	transport = gzhttp.Transport(transport)
	config.Transport = transport
}

// SelectObjectOpts - opts entered for select API
type SelectObjectOpts struct {
	InputSerOpts    map[string]map[string]string
	OutputSerOpts   map[string]map[string]string
	CompressionType minio.SelectCompressionType
}

type headerTransport struct {
	http.RoundTripper
	customHeader http.Header
}

func (h *headerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	maps.Copy(request.Header, h.customHeader)
	return h.RoundTripper.RoundTrip(request)
}
