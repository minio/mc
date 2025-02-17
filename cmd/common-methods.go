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
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/pkg/v3/env"
)

// Check if the passed URL represents a folder. It may or may not exist yet.
// If it exists, we can easily check if it is a folder, if it doesn't exist,
// we can guess if the url is a folder from how it looks.
func isAliasURLDir(ctx context.Context, aliasURL string, keys map[string][]prefixSSEPair, timeRef time.Time, ignoreBucketExists bool) (bool, *ClientContent) {
	// If the target url exists, check if it is a directory
	// and return immediately.
	_, targetContent, err := url2Stat(ctx, url2StatOptions{
		urlStr:                  aliasURL,
		versionID:               "",
		fileAttr:                false,
		encKeyDB:                keys,
		timeRef:                 timeRef,
		isZip:                   false,
		ignoreBucketExistsCheck: ignoreBucketExists,
	})
	if err == nil {
		return targetContent.Type.IsDir(), targetContent
	}

	_, expandedURL, _ := mustExpandAlias(aliasURL)

	// Check if targetURL is an FS or S3 aliased url
	if expandedURL == aliasURL {
		// This is an FS url, check if the url has a separator at the end
		return strings.HasSuffix(aliasURL, string(filepath.Separator)), targetContent
	}

	// This is an S3 url, then:
	//   *) If alias format is specified, return false
	//   *) If alias/bucket is specified, return true
	//   *) If alias/bucket/prefix, check if prefix has
	//       has a trailing slash.
	pathURL := filepath.ToSlash(aliasURL)
	fields := strings.Split(pathURL, "/")
	switch len(fields) {
	// Nothing or alias format
	case 0, 1:
		return false, targetContent
	// alias/bucket format
	case 2:
		return true, targetContent
	} // default case..

	// alias/bucket/prefix format
	return strings.HasSuffix(pathURL, "/"), targetContent
}

// getSourceStreamMetadataFromURL gets a reader from URL.
func getSourceStreamMetadataFromURL(ctx context.Context, aliasedURL, versionID string, timeRef time.Time, encKeyDB map[string][]prefixSSEPair, zip bool) (reader io.ReadCloser,
	content *ClientContent, err *probe.Error,
) {
	alias, urlStrFull, _, err := expandAlias(aliasedURL)
	if err != nil {
		return nil, nil, err.Trace(aliasedURL)
	}
	if !timeRef.IsZero() {
		_, content, err := url2Stat(ctx, url2StatOptions{urlStr: aliasedURL, versionID: "", fileAttr: false, encKeyDB: nil, timeRef: timeRef, isZip: false, ignoreBucketExistsCheck: false})
		if err != nil {
			return nil, nil, err
		}
		versionID = content.VersionID
	}
	return getSourceStream(ctx, alias, urlStrFull, getSourceOpts{
		GetOptions: GetOptions{
			SSE:       getSSE(aliasedURL, encKeyDB[alias]),
			VersionID: versionID,
			Zip:       zip,
		},
	})
}

type getSourceOpts struct {
	GetOptions
	preserve bool
}

// getSourceStreamFromURL gets a reader from URL.
func getSourceStreamFromURL(ctx context.Context, urlStr string, encKeyDB map[string][]prefixSSEPair, opts getSourceOpts) (reader io.ReadCloser, err *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	opts.SSE = getSSE(urlStr, encKeyDB[alias])
	reader, _, err = getSourceStream(ctx, alias, urlStrFull, opts)
	return reader, err
}

// Verify if reader is a generic ReaderAt
func isReadAt(reader io.Reader) (ok bool) {
	var v *os.File
	v, ok = reader.(*os.File)
	if ok {
		// Stdin, Stdout and Stderr all have *os.File type
		// which happen to also be io.ReaderAt compatible
		// we need to add special conditions for them to
		// be ignored by this function.
		for _, f := range []string{
			"/dev/stdin",
			"/dev/stdout",
			"/dev/stderr",
		} {
			if f == v.Name() {
				ok = false
				break
			}
		}
	}
	return
}

// getSourceStream gets a reader from URL.
func getSourceStream(ctx context.Context, alias, urlStr string, opts getSourceOpts) (reader io.ReadCloser, content *ClientContent, err *probe.Error) {
	sourceClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}

	reader, content, err = sourceClnt.Get(ctx, opts.GetOptions)
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}

	return reader, content, nil
}

// putTargetRetention sets retention headers if any
func putTargetRetention(ctx context.Context, alias, urlStr string, metadata map[string]string) *probe.Error {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return err.Trace(alias, urlStr)
	}
	lockModeStr, ok := metadata[AmzObjectLockMode]
	lockMode := minio.RetentionMode("")
	if ok {
		lockMode = minio.RetentionMode(lockModeStr)
		delete(metadata, AmzObjectLockMode)
	}

	retainUntilDateStr, ok := metadata[AmzObjectLockRetainUntilDate]
	retainUntilDate := timeSentinel
	if ok {
		delete(metadata, AmzObjectLockRetainUntilDate)
		if t, e := time.Parse(time.RFC3339, retainUntilDateStr); e == nil {
			retainUntilDate = t.UTC()
		}
	}
	if err := targetClnt.PutObjectRetention(ctx, "", lockMode, retainUntilDate, false); err != nil {
		return err.Trace(alias, urlStr)
	}
	return nil
}

// putTargetStream writes to URL from Reader.
func putTargetStream(ctx context.Context, alias, urlStr, mode, until, legalHold string, reader io.Reader, size int64, progress io.Reader, opts PutOptions) (int64, *probe.Error) {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}

	if mode != "" {
		opts.metadata[AmzObjectLockMode] = mode
	}
	if until != "" {
		opts.metadata[AmzObjectLockRetainUntilDate] = until
	}
	if legalHold != "" {
		opts.metadata[AmzObjectLockLegalHold] = legalHold
	}

	n, err := targetClnt.Put(ctx, reader, size, progress, opts)
	if err != nil {
		return n, err.Trace(alias, urlStr)
	}
	return n, nil
}

// putTargetStreamWithURL writes to URL from reader. If length=-1, read until EOF.
func putTargetStreamWithURL(urlStr string, reader io.Reader, size int64, opts PutOptions) (int64, *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}
	contentType := guessURLContentType(urlStr)
	if opts.metadata == nil {
		opts.metadata = map[string]string{}
	}
	opts.metadata["Content-Type"] = contentType
	return putTargetStream(context.Background(), alias, urlStrFull, "", "", "", reader, size, nil, opts)
}

// copySourceToTargetURL copies to targetURL from source.
func copySourceToTargetURL(ctx context.Context, alias, urlStr, source, sourceVersionID, mode, until, legalHold string, size int64, progress io.Reader, opts CopyOptions) *probe.Error {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return err.Trace(alias, urlStr)
	}

	opts.versionID = sourceVersionID
	opts.size = size
	opts.metadata[AmzObjectLockMode] = mode
	opts.metadata[AmzObjectLockRetainUntilDate] = until
	opts.metadata[AmzObjectLockLegalHold] = legalHold

	err = targetClnt.Copy(ctx, source, opts, progress)
	if err != nil {
		return err.Trace(alias, urlStr)
	}
	return nil
}

func filterMetadata(metadata map[string]string) map[string]string {
	newMetadata := map[string]string{}
	for k, v := range metadata {
		if httpguts.ValidHeaderFieldName(k) && httpguts.ValidHeaderFieldValue(v) {
			newMetadata[k] = v
		}
	}
	for k := range metadata {
		if strings.HasPrefix(http.CanonicalHeaderKey(k), http.CanonicalHeaderKey(serverEncryptionKeyPrefix)) {
			delete(newMetadata, k)
		}
	}
	return newMetadata
}

// getAllMetadata - returns a map of user defined function
// by combining the usermetadata of object and values passed by attr keyword
func getAllMetadata(ctx context.Context, sourceAlias, sourceURLStr string, srcSSE encrypt.ServerSide, urls URLs) (map[string]string, *probe.Error) {
	metadata := make(map[string]string)
	sourceClnt, err := newClientFromAlias(sourceAlias, sourceURLStr)
	if err != nil {
		return nil, err.Trace(sourceAlias, sourceURLStr)
	}

	st, err := sourceClnt.Stat(ctx, StatOptions{preserve: true, sse: srcSSE})
	if err != nil {
		return nil, err.Trace(sourceAlias, sourceURLStr)
	}

	for k, v := range st.Metadata {
		metadata[http.CanonicalHeaderKey(k)] = v
	}

	for k, v := range urls.TargetContent.UserMetadata {
		metadata[http.CanonicalHeaderKey(k)] = v
	}

	return filterMetadata(metadata), nil
}

// uploadSourceToTargetURL - uploads to targetURL from source.
// optionally optimizes copy for object sizes <= 5GiB by using
// server side copy operation.
func uploadSourceToTargetURL(ctx context.Context, uploadOpts uploadSourceToTargetURLOpts) URLs {
	sourceAlias := uploadOpts.urls.SourceAlias
	sourceURL := uploadOpts.urls.SourceContent.URL
	sourceVersion := uploadOpts.urls.SourceContent.VersionID
	targetAlias := uploadOpts.urls.TargetAlias
	targetURL := uploadOpts.urls.TargetContent.URL
	length := uploadOpts.urls.SourceContent.Size
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, uploadOpts.urls.SourceContent.URL.Path))
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, uploadOpts.urls.TargetContent.URL.Path))

	srcSSE := getSSE(sourcePath, uploadOpts.encKeyDB[sourceAlias])
	tgtSSE := getSSE(targetPath, uploadOpts.encKeyDB[targetAlias])

	var err *probe.Error
	metadata := map[string]string{}
	var mode, until, legalHold string

	// add object retention fields in metadata for target, if target wants
	// to override defaults from source, usually happens in `cp` command.
	// for the most part source metadata is copied over.
	if uploadOpts.urls.TargetContent.RetentionEnabled {
		m := minio.RetentionMode(strings.ToUpper(uploadOpts.urls.TargetContent.RetentionMode))
		if !m.IsValid() {
			return uploadOpts.urls.WithError(probe.NewError(errors.New("invalid retention mode")).Trace(targetURL.String()))
		}

		var dur uint64
		var unit minio.ValidityUnit
		dur, unit, err = parseRetentionValidity(uploadOpts.urls.TargetContent.RetentionDuration)
		if err != nil {
			return uploadOpts.urls.WithError(err.Trace(targetURL.String()))
		}

		mode = uploadOpts.urls.TargetContent.RetentionMode

		until, err = getRetainUntilDate(dur, unit)
		if err != nil {
			return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
		}
	}

	// add object legal hold fields in metadata for target, if target wants
	// to override defaults from source, usually happens in `cp` command.
	// for the most part source metadata is copied over.
	if uploadOpts.urls.TargetContent.LegalHoldEnabled {
		switch minio.LegalHoldStatus(uploadOpts.urls.TargetContent.LegalHold) {
		case minio.LegalHoldDisabled:
		case minio.LegalHoldEnabled:
		default:
			return uploadOpts.urls.WithError(errInvalidArgument().Trace(uploadOpts.urls.TargetContent.LegalHold))
		}
		legalHold = uploadOpts.urls.TargetContent.LegalHold
	}

	for k, v := range uploadOpts.urls.SourceContent.UserMetadata {
		metadata[http.CanonicalHeaderKey(k)] = v
	}
	for k, v := range uploadOpts.urls.SourceContent.Metadata {
		metadata[http.CanonicalHeaderKey(k)] = v
	}

	// Optimize for server side copy if the host is same.
	if sourceAlias == targetAlias && !uploadOpts.isZip && !uploadOpts.urls.checksum.IsSet() {
		// preserve new metadata and save existing ones.
		if uploadOpts.preserve {
			currentMetadata, err := getAllMetadata(ctx, sourceAlias, sourceURL.String(), srcSSE, uploadOpts.urls)
			if err != nil {
				return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
			}
			for k, v := range currentMetadata {
				metadata[k] = v
			}
		}

		// Get metadata from target content as well
		for k, v := range uploadOpts.urls.TargetContent.Metadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		// Get userMetadata from target content as well
		for k, v := range uploadOpts.urls.TargetContent.UserMetadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		sourcePath := filepath.ToSlash(sourceURL.Path)
		if uploadOpts.urls.SourceContent.RetentionEnabled {
			err = putTargetRetention(ctx, targetAlias, targetURL.String(), metadata)
			return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
		}

		opts := CopyOptions{
			srcSSE:           srcSSE,
			tgtSSE:           tgtSSE,
			metadata:         filterMetadata(metadata),
			disableMultipart: uploadOpts.urls.DisableMultipart,
			isPreserve:       uploadOpts.preserve,
			storageClass:     uploadOpts.urls.TargetContent.StorageClass,
		}

		err = copySourceToTargetURL(ctx, targetAlias, targetURL.String(), sourcePath, sourceVersion, mode, until,
			legalHold, length, uploadOpts.progress, opts)
	} else {
		if uploadOpts.urls.SourceContent.RetentionEnabled {
			// preserve new metadata and save existing ones.
			if uploadOpts.preserve {
				currentMetadata, err := getAllMetadata(ctx, sourceAlias, sourceURL.String(), srcSSE, uploadOpts.urls)
				if err != nil {
					return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
				}
				for k, v := range currentMetadata {
					metadata[k] = v
				}
			}

			// Get metadata from target content as well
			for k, v := range uploadOpts.urls.TargetContent.Metadata {
				metadata[http.CanonicalHeaderKey(k)] = v
			}

			// Get userMetadata from target content as well
			for k, v := range uploadOpts.urls.TargetContent.UserMetadata {
				metadata[http.CanonicalHeaderKey(k)] = v
			}

			err = putTargetRetention(ctx, targetAlias, targetURL.String(), metadata)
			return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
		}

		// Proceed with regular stream copy.
		var (
			content *ClientContent
			reader  io.ReadCloser
		)

		reader, content, err = getSourceStream(ctx, sourceAlias, sourceURL.String(), getSourceOpts{
			GetOptions: GetOptions{
				VersionID: sourceVersion,
				SSE:       srcSSE,
				Zip:       uploadOpts.isZip,
				Preserve:  uploadOpts.preserve,
			},
		})
		if err != nil {
			return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
		}
		defer reader.Close()

		if uploadOpts.updateProgressTotal {
			pg, ok := uploadOpts.progress.(*progressBar)
			if ok {
				pg.SetTotal(content.Size)
			}
		}

		metadata := make(map[string]string, len(content.Metadata))
		for k, v := range content.Metadata {
			metadata[k] = v
		}

		// Get metadata from target content as well
		for k, v := range uploadOpts.urls.TargetContent.Metadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		// Get userMetadata from target content as well
		for k, v := range uploadOpts.urls.TargetContent.UserMetadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		if content.Tags != nil {
			tags, err := url.PathUnescape(s3utils.TagEncode(content.Tags))
			if err != nil {
				return uploadOpts.urls.WithError(probe.NewError(err))
			}
			metadata["X-Amz-Tagging"] = tags
			delete(metadata, "X-Amz-Tagging-Count")
		}

		var e error
		var multipartSize uint64
		var multipartThreads int
		var v string
		if uploadOpts.multipartSize == "" {
			v = env.Get("MC_UPLOAD_MULTIPART_SIZE", "")
		} else {
			v = uploadOpts.multipartSize
		}
		if v != "" {
			multipartSize, e = humanize.ParseBytes(v)
			if e != nil {
				return uploadOpts.urls.WithError(probe.NewError(e))
			}
		}

		if uploadOpts.multipartThreads == "" {
			multipartThreads, e = strconv.Atoi(env.Get("MC_UPLOAD_MULTIPART_THREADS", "4"))
		} else {
			multipartThreads, e = strconv.Atoi(uploadOpts.multipartThreads)
		}
		if e != nil {
			return uploadOpts.urls.WithError(probe.NewError(e))
		}

		putOpts := PutOptions{
			metadata:         filterMetadata(metadata),
			sse:              tgtSSE,
			storageClass:     uploadOpts.urls.TargetContent.StorageClass,
			md5:              uploadOpts.urls.MD5,
			disableMultipart: uploadOpts.urls.DisableMultipart,
			isPreserve:       uploadOpts.preserve,
			multipartSize:    multipartSize,
			multipartThreads: uint(multipartThreads),
			ifNotExists:      uploadOpts.ifNotExists,
			checksum:         uploadOpts.urls.checksum,
		}

		if isReadAt(reader) || length == 0 {
			_, err = putTargetStream(ctx, targetAlias, targetURL.String(), mode, until,
				legalHold, reader, length, uploadOpts.progress, putOpts)
		} else {
			_, err = putTargetStream(ctx, targetAlias, targetURL.String(), mode, until,
				legalHold, io.LimitReader(reader, length), length, uploadOpts.progress, putOpts)
		}
	}
	if err != nil {
		return uploadOpts.urls.WithError(err.Trace(sourceURL.String()))
	}

	return uploadOpts.urls.WithError(nil)
}

// newClientFromAlias gives a new client interface for matching
// alias entry in the mc config file. If no matching host config entry
// is found, fs client is returned.
func newClientFromAlias(alias, urlStr string) (Client, *probe.Error) {
	alias, _, hostCfg, err := expandAlias(alias)
	if err != nil {
		return nil, err.Trace(alias, urlStr)
	}

	if hostCfg == nil {
		// No matching host config. So we treat it like a
		// filesystem.
		fsClient, fsErr := fsNew(urlStr)
		if fsErr != nil {
			return nil, fsErr.Trace(alias, urlStr)
		}
		return fsClient, nil
	}

	s3Config := NewS3Config(alias, urlStr, hostCfg)
	s3Client, err := S3New(s3Config)
	if err != nil {
		return nil, err.Trace(alias, urlStr)
	}
	return s3Client, nil
}

// urlRgx - verify if aliased url is real URL.
var urlRgx = regexp.MustCompile("^https?://")

// newClient gives a new client interface
func newClient(aliasedURL string) (Client, *probe.Error) {
	alias, urlStrFull, hostCfg, err := expandAlias(aliasedURL)
	if err != nil {
		return nil, err.Trace(aliasedURL)
	}
	// Verify if the aliasedURL is a real URL, fail in those cases
	// indicating the user to add alias.
	if hostCfg == nil && urlRgx.MatchString(aliasedURL) {
		return nil, errInvalidAliasedURL(aliasedURL).Trace(aliasedURL)
	}
	return newClientFromAlias(alias, urlStrFull)
}

// ParseForm parses a http.Request form and populates the array
func ParseForm(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	for k, v := range r.PostForm {
		if _, ok := r.Form[k]; !ok {
			r.Form[k] = v
		}
	}
	return nil
}

type uploadSourceToTargetURLOpts struct {
	urls                URLs
	progress            io.Reader
	encKeyDB            map[string][]prefixSSEPair
	preserve, isZip     bool
	multipartSize       string
	multipartThreads    string
	updateProgressTotal bool
	ifNotExists         bool
}
