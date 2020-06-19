/*
 * MinIO Client (C) 2015-2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this fs except in compliance with the License.
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
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/mc/cmd/ilm"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/encrypt"
	"github.com/minio/minio-go/v6/pkg/tags"
)

type snapClient struct {
	PathURL *ClientURL

	snapName string
	s3Target Client
}

// snapNew - instantiate a new snapshot
func snapNew(snapName, spath string) (Client, *probe.Error) {
	f, err := openSnapshotFile(filepath.Join(snapName, "metadata.json"))
	if err != nil {
		return &snapClient{}, err
	}

	metadataBytes, e := ioutil.ReadAll(f)
	if e != nil {
		return &snapClient{}, probe.NewError(e)
	}

	var s3Target S3Target
	e = json.Unmarshal(metadataBytes, &s3Target)
	if e != nil {
		return &snapClient{}, probe.NewError(e)
	}

	pathURL := strings.TrimPrefix(spath, snapName)

	hostCfg := hostConfigV9(s3Target)

	u, e := url.Parse(s3Target.URL)
	if e != nil {
		return nil, probe.NewError(e)
	}
	u.Path = path.Join(u.Path, pathURL)

	s3Config := NewS3Config(u.String(), &hostCfg)
	clnt, err := S3New(s3Config)
	if err != nil {
		return nil, err
	}

	return &snapClient{
		PathURL:  newClientURL(normalizePath(pathURL)),
		snapName: snapName,
		s3Target: clnt,
	}, nil
}

// URL get url.
func (s *snapClient) GetURL() ClientURL {
	return *s.PathURL
}

// Select replies a stream of query results.
func (s *snapClient) Select(ctx context.Context, expression string, sse encrypt.ServerSide, opts SelectObjectOpts) (io.ReadCloser, *probe.Error) {
	return nil, probe.NewError(APINotImplemented{
		API:     "Select",
		APIType: "snapshot",
	})
}

func (s *snapClient) Watch(ctx context.Context, options WatchOptions) (*WatchObject, *probe.Error) {
	return nil, probe.NewError(APINotImplemented{
		API:     "Watch",
		APIType: "snapshot",
	})
}

/// Object operations.

func (s *snapClient) Put(ctx context.Context, reader io.Reader, size int64, metadata map[string]string, progress io.Reader, sse encrypt.ServerSide, md5, disableMultipart bool) (int64, *probe.Error) {
	return 0, probe.NewError(APINotImplemented{
		API:     "Put",
		APIType: "snapshot",
	})
}

func (s *snapClient) ShareDownload(ctx context.Context, expires time.Duration) (string, *probe.Error) {
	return "", probe.NewError(APINotImplemented{
		API:     "ShareDownload",
		APIType: "snapshot",
	})
}

func (s *snapClient) ShareUpload(startsWith bool, expires time.Duration, contentType string) (string, map[string]string, *probe.Error) {
	return "", nil, probe.NewError(APINotImplemented{
		API:     "ShareUpload",
		APIType: "snapshot",
	})
}

// Copy - copy data from source to destination
func (s *snapClient) Copy(ctx context.Context, source string, size int64, progress io.Reader, srcSSE, tgtSSE encrypt.ServerSide, metadata map[string]string, disableMultipart bool) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "Copy",
		APIType: "snapshot",
	})
}

func (s *snapClient) Get(ctx context.Context, sse encrypt.ServerSide) (io.ReadCloser, *probe.Error) {
	return s.GetWithOptions(ctx, GetOptions{sse: sse})
}

// Get returns reader and any additional metadata.
func (s *snapClient) GetWithOptions(ctx context.Context, opts GetOptions) (io.ReadCloser, *probe.Error) {
	return s.s3Target.GetWithOptions(ctx, opts)
}

// Remove - remove entry read from clientContent channel.
func (s *snapClient) Remove(ctx context.Context, isIncomplete, isRemoveBucket, isBypass bool, contentCh <-chan *ClientContent) <-chan *probe.Error {
	errorCh := make(chan *probe.Error, 1)
	defer close(errorCh)
	errorCh <- probe.NewError(APINotImplemented{
		API:     "Remove",
		APIType: "snapshot",
	})
	return errorCh
}

func (s *snapClient) Snapshot(ctx context.Context, timeRef time.Time) <-chan *ClientContent {
	contentCh := make(chan *ClientContent, 1)
	contentCh <- &ClientContent{Err: probe.NewError(APINotImplemented{
		API:     "Snapshot",
		APIType: "snapshot",
	})}
	close(contentCh)
	return contentCh
}

// url2BucketAndObject gives bucketName and objectName from URL path.
func (s *snapClient) url2BucketAndObject() (bucketName, objectName string) {
	path := s.PathURL.Path
	tokens := splitStr(path, string(s.PathURL.Separator), 3)
	return tokens[1], tokens[2]
}

func (s *snapClient) List(ctx context.Context, isRecursive, _, _ bool, showDir DirOpt) <-chan *ClientContent {
	contentCh := make(chan *ClientContent)
	go s.list(ctx, contentCh, isRecursive, false, false, showDir)
	return contentCh
}

func (s *snapClient) getBucketContents(ctx context.Context, bucket string, contentCh chan *ClientContent, filter func(SnapshotEntry) (SnapshotEntry, filterAction)) {
	f, err := openSnapshotFile(filepath.Join(s.snapName, "buckets", bucket))
	if err != nil {
		contentCh <- &ClientContent{Err: err}
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var (
			entry  SnapshotEntry
			action filterAction
		)
		_, e := entry.UnmarshalMsg(line)
		if e != nil {
			contentCh <- &ClientContent{Err: probe.NewError(e)}
			return
		}

		if filter != nil {
			entry, action = filter(entry)
			if action == filterSkipEntry {
				continue
			}
		}

		url := s.PathURL.Clone()
		url.Path = path.Join("/", bucket, entry.Key)

		var mod os.FileMode
		if entry.Key == "" || strings.HasSuffix(entry.Key, "/") {
			mod ^= os.ModeDir
		}

		c := &ClientContent{
			URL:            url,
			Type:           mod,
			VersionID:      entry.VersionID,
			Size:           entry.Size,
			Time:           entry.ModTime,
			ETag:           entry.ETag,
			StorageClass:   entry.StorageClass,
			IsDeleteMarker: entry.IsDeleteMarker,
			IsLatest:       entry.IsLatest,
		}
		contentCh <- c

		if action == filterAbort {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		contentCh <- &ClientContent{Err: probe.NewError(err)}
		return
	}

}

func (s *snapClient) listBuckets(ctx context.Context, contentCh chan *ClientContent, isRecursive, _, _ bool, showDir DirOpt) {
	buckets, err := listSnapshotBuckets(s.snapName)
	if err != nil {
		contentCh <- &ClientContent{Err: err}
		return
	}

	if !isRecursive {
		for _, b := range buckets {
			url := s.PathURL.Clone()
			url.Path = path.Join("/", b)

			c := &ClientContent{
				URL:  url,
				Type: os.ModeDir,
			}

			contentCh <- c
		}
		return
	}

	filter := func(entry SnapshotEntry) (SnapshotEntry, filterAction) {
		if entry.IsDeleteMarker {
			return SnapshotEntry{}, filterSkipEntry
		}
		return entry, filterNoAction
	}

	for _, b := range buckets {
		s.getBucketContents(ctx, b, contentCh, filter)
	}
}

// List - list files and folders.
func (s *snapClient) list(ctx context.Context, contentCh chan *ClientContent, isRecursive, _, _ bool, showDir DirOpt) {
	defer close(contentCh)

	bucket, prefix := s.url2BucketAndObject()
	if bucket == "" {
		s.listBuckets(ctx, contentCh, isRecursive, false, false, showDir)
		return
	}

	var lastKey string

	filter := func(entry SnapshotEntry) (SnapshotEntry, filterAction) {
		if !strings.HasPrefix(entry.Key, prefix) {
			return SnapshotEntry{}, filterSkipEntry
		}
		if entry.IsDeleteMarker {
			return SnapshotEntry{}, filterSkipEntry
		}
		if !isRecursive {
			tmpKey := strings.TrimPrefix(entry.Key, prefix)
			idx := strings.Index(tmpKey, "/")
			if idx > 0 {
				entry.Key = tmpKey[:len(prefix)+idx+1]
			}
			if entry.Key == lastKey {
				return SnapshotEntry{}, filterSkipEntry
			}
			lastKey = entry.Key
		}
		return entry, filterNoAction
	}

	s.getBucketContents(ctx, bucket, contentCh, filter)
}

// MakeBucket - create a new bucket.
func (s *snapClient) MakeBucket(ctx context.Context, region string, ignoreExisting, withLock bool) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "MakeBucket",
		APIType: "snapshot",
	})
}

// Set object lock configuration of bucket.
func (s *snapClient) SetObjectLockConfig(ctx context.Context, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "SetObjectLockConfig",
		APIType: "snapshot",
	})
}

// Get object lock configuration of bucket.
func (s *snapClient) GetObjectLockConfig(ctx context.Context) (mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, err *probe.Error) {
	return "", 0, "", probe.NewError(APINotImplemented{
		API:     "GetObjectLockConfig",
		APIType: "snapshot",
	})
}

// GetAccessRules - unsupported API
func (s *snapClient) GetAccessRules(ctx context.Context) (map[string]string, *probe.Error) {
	return map[string]string{}, probe.NewError(APINotImplemented{
		API:     "GetBucketPolicy",
		APIType: "snapshot",
	})
}

// Set object retention for a given object.
func (s *snapClient) PutObjectRetention(ctx context.Context, mode minio.RetentionMode, retainUntilDate time.Time, bypassGovernance bool) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "PutObjectRetention",
		APIType: "snapshot",
	})
}

// Set object legal hold for a given object.
func (s *snapClient) PutObjectLegalHold(ctx context.Context, lhold minio.LegalHoldStatus) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "PutObjectLegalHold",
		APIType: "snapshot",
	})
}

// GetAccess - get access policy permissions.
func (s *snapClient) GetAccess(ctx context.Context) (access string, policyJSON string, err *probe.Error) {
	return "", "", probe.NewError(APINotImplemented{
		API:     "GetAccess",
		APIType: "snapshot",
	})

}

// SetAccess - set access policy permissions.
func (s *snapClient) SetAccess(ctx context.Context, access string, isJSON bool) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "SetAccess",
		APIType: "snapshot",
	})
}

func (s *snapClient) Stat(ctx context.Context, _, _ bool, sse encrypt.ServerSide) (content *ClientContent, err *probe.Error) {
	return s.StatWithOptions(ctx, false, false, StatOptions{sse: sse})
}

type filterAction int

const (
	filterNoAction filterAction = iota
	filterSkipEntry
	filterAbort
)

func (s *snapClient) statBucket(ctx context.Context, bucket string) (content *ClientContent, err *probe.Error) {
	contentCh := make(chan *ClientContent, 1)

	filter := func(entry SnapshotEntry) (SnapshotEntry, filterAction) {
		if entry.IsDeleteMarker {
			return SnapshotEntry{}, filterSkipEntry
		}
		return SnapshotEntry{}, filterAbort
	}

	s.getBucketContents(ctx, bucket, contentCh, filter)

	content = <-contentCh
	if content == nil {
		return nil, probe.NewError(BucketDoesNotExist{Bucket: bucket})
	}

	if content.Err != nil {
		return nil, content.Err
	}

	return content, nil
}

// Stat - get metadata from path.
func (s *snapClient) StatWithOptions(ctx context.Context, _, _ bool, opts StatOptions) (content *ClientContent, err *probe.Error) {
	bucket, object := s.url2BucketAndObject()

	if bucket == "" {
		return nil, probe.NewError(BucketNameEmpty{})
	}

	if object == "" {
		return s.statBucket(ctx, bucket)
	}

	object = strings.TrimSuffix(object, "/")

	filter := func(entry SnapshotEntry) (SnapshotEntry, filterAction) {
		if entry.IsDeleteMarker {
			return SnapshotEntry{}, filterSkipEntry
		}
		if strings.HasPrefix(entry.Key, object+"/") {
			return SnapshotEntry{Key: object + "/"}, filterAbort
		}
		if entry.Key != object {
			return SnapshotEntry{}, filterSkipEntry
		}
		return entry, filterAbort
	}

	contentCh := make(chan *ClientContent, 1)

	s.getBucketContents(ctx, bucket, contentCh, filter)

	content = <-contentCh
	if content == nil {
		return nil, probe.NewError(ObjectMissing{})
	}

	if content.Err != nil {
		return nil, content.Err
	}

	return content, nil
}

func (s *snapClient) AddUserAgent(_, _ string) {
}

// Get Object Tags
func (s *snapClient) GetTags(ctx context.Context) (*tags.Tags, *probe.Error) {
	return nil, probe.NewError(APINotImplemented{
		API:     "GetObjectTagging",
		APIType: "snapshot",
	})
}

// Set Object tags
func (s *snapClient) SetTags(ctx context.Context, tags string) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "SetObjectTagging",
		APIType: "snapshot",
	})
}

// Delete object tags
func (s *snapClient) DeleteTags(ctx context.Context) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "DeleteObjectTagging",
		APIType: "snapshot",
	})
}

// Get lifecycle configuration for a given bucket, not implemented.
func (s *snapClient) GetLifecycle(ctx context.Context) (ilm.LifecycleConfiguration, *probe.Error) {
	return ilm.LifecycleConfiguration{}, probe.NewError(APINotImplemented{
		API:     "GetLifecycle",
		APIType: "snapshot",
	})
}

// Set lifecycle configuration for a given bucket, not implemented.
func (s *snapClient) SetLifecycle(ctx context.Context, _ ilm.LifecycleConfiguration) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "SetLifecycle",
		APIType: "snapshot",
	})
}
