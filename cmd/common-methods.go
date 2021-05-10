// Copyright (c) 2015-2021 MinIO, Inc.
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
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"
	"gopkg.in/h2non/filetype.v1"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// decode if the key is encoded key and returns the key
func getDecodedKey(sseKeys string) (key string, err *probe.Error) {
	keyString := ""
	for i, sse := range strings.Split(sseKeys, ",") {
		if i > 0 {
			keyString = keyString + ","
		}
		sseString, err := parseKey(sse)
		if err != nil {
			return "", err
		}
		keyString = keyString + sseString
	}
	return keyString, nil
}

// Validate the key
func parseKey(sseKeys string) (sse string, err *probe.Error) {
	encryptString := strings.SplitN(sseKeys, "=", 2)
	if len(encryptString) < 2 {
		return "", probe.NewError(errors.New("SSE-C prefix should be of the form prefix1=key1,... "))
	}

	secretValue := encryptString[1]
	if len(secretValue) == 32 {
		return sseKeys, nil
	}
	decodedString, e := base64.StdEncoding.DecodeString(secretValue)
	if e != nil || len(decodedString) != 32 {
		return "", probe.NewError(errors.New("Encryption key should be 32 bytes plain text key or 44 bytes base64 encoded key"))
	}
	return encryptString[0] + "=" + string(decodedString), nil
}

// parse and return encryption key pairs per alias.
func getEncKeys(ctx *cli.Context) (map[string][]prefixSSEPair, *probe.Error) {
	sseServer := os.Getenv("MC_ENCRYPT")
	if prefix := ctx.String("encrypt"); prefix != "" {
		sseServer = prefix
	}

	sseKeys := os.Getenv("MC_ENCRYPT_KEY")
	if keyPrefix := ctx.String("encrypt-key"); keyPrefix != "" {
		if sseServer != "" && strings.Contains(keyPrefix, sseServer) {
			return nil, errConflictSSE(sseServer, keyPrefix).Trace(ctx.Args()...)
		}
		sseKeys = keyPrefix
	}
	var err *probe.Error
	if sseKeys != "" {
		sseKeys, err = getDecodedKey(sseKeys)
		if err != nil {
			return nil, err.Trace(sseKeys)
		}
	}

	encKeyDB, err := parseAndValidateEncryptionKeys(sseKeys, sseServer)
	if err != nil {
		return nil, err.Trace(sseKeys)
	}

	return encKeyDB, nil
}

// Check if the passed URL represents a folder. It may or may not exist yet.
// If it exists, we can easily check if it is a folder, if it doesn't exist,
// we can guess if the url is a folder from how it looks.
func isAliasURLDir(ctx context.Context, aliasURL string, keys map[string][]prefixSSEPair, timeRef time.Time) bool {
	// If the target url exists, check if it is a directory
	// and return immediately.
	_, targetContent, err := url2Stat(ctx, aliasURL, "", false, keys, timeRef)
	if err == nil {
		return targetContent.Type.IsDir()
	}

	_, expandedURL, _ := mustExpandAlias(aliasURL)

	// Check if targetURL is an FS or S3 aliased url
	if expandedURL == aliasURL {
		// This is an FS url, check if the url has a separator at the end
		return strings.HasSuffix(aliasURL, string(filepath.Separator))
	}

	// This is an S3 url, then:
	//   *) If alias format is specified, return false
	//   *) If alias/bucket is specified, return true
	//   *) If alias/bucket/prefix, check if prefix has
	//	     has a trailing slash.
	pathURL := filepath.ToSlash(aliasURL)
	fields := strings.Split(pathURL, "/")
	switch len(fields) {
	// Nothing or alias format
	case 0, 1:
		return false
	// alias/bucket format
	case 2:
		return true
	} // default case..

	// alias/bucket/prefix format
	return strings.HasSuffix(pathURL, "/")
}

// getSourceStreamMetadataFromURL gets a reader from URL.
func getSourceStreamMetadataFromURL(ctx context.Context, aliasedURL, versionID string, timeRef time.Time, encKeyDB map[string][]prefixSSEPair) (reader io.ReadCloser,
	metadata map[string]string, err *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(aliasedURL)
	if err != nil {
		return nil, nil, err.Trace(aliasedURL)
	}
	if !timeRef.IsZero() {
		_, content, err := url2Stat(ctx, aliasedURL, "", false, nil, timeRef)
		if err != nil {
			return nil, nil, err
		}
		versionID = content.VersionID
	}
	sseKey := getSSE(aliasedURL, encKeyDB[alias])
	return getSourceStream(ctx, alias, urlStrFull, versionID, true, sseKey, false)
}

// getSourceStreamFromURL gets a reader from URL.
func getSourceStreamFromURL(ctx context.Context, urlStr, versionID string, encKeyDB map[string][]prefixSSEPair) (reader io.ReadCloser, err *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	sse := getSSE(urlStr, encKeyDB[alias])
	reader, _, err = getSourceStream(ctx, alias, urlStrFull, versionID, false, sse, false)
	return reader, err
}

func probeContentType(reader io.Reader) (ctype string, err *probe.Error) {
	ctype = "application/octet-stream"
	// Read a chunk to decide between utf-8 text and binary
	if s, ok := reader.(io.Seeker); ok {
		var buf [512]byte
		n, _ := io.ReadFull(reader, buf[:])
		if n <= 0 {
			return ctype, nil
		}
		kind, e := filetype.Match(buf[:n])
		if e != nil {
			return ctype, probe.NewError(e)
		}
		// rewind to output whole file
		if _, e = s.Seek(0, io.SeekStart); e != nil {
			return ctype, probe.NewError(e)
		}
		if kind.MIME.Value != "" {
			ctype = kind.MIME.Value
		}
	}
	return ctype, nil
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
func getSourceStream(ctx context.Context, alias, urlStr, versionID string, fetchStat bool, sse encrypt.ServerSide, preserve bool) (reader io.ReadCloser, metadata map[string]string, err *probe.Error) {
	sourceClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}
	reader, err = sourceClnt.Get(ctx, GetOptions{SSE: sse, VersionID: versionID})
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}

	metadata = make(map[string]string)
	if fetchStat {
		var st *ClientContent
		mo, mok := reader.(*minio.Object)
		if mok {
			oinfo, e := mo.Stat()
			if e != nil {
				return nil, nil, probe.NewError(e).Trace(alias, urlStr)
			}
			st = &ClientContent{}
			st.Time = oinfo.LastModified
			st.Size = oinfo.Size
			st.ETag = oinfo.ETag
			st.Expires = oinfo.Expires
			st.Type = os.FileMode(0664)
			st.Metadata = map[string]string{}
			for k := range oinfo.Metadata {
				st.Metadata[k] = oinfo.Metadata.Get(k)
			}
			st.ETag = oinfo.ETag
		} else {
			st, err = sourceClnt.Stat(ctx, StatOptions{preserve: preserve, sse: sse})
			if err != nil {
				return nil, nil, err.Trace(alias, urlStr)
			}
		}

		for k, v := range st.Metadata {
			if httpguts.ValidHeaderFieldName(k) &&
				httpguts.ValidHeaderFieldValue(v) {
				metadata[k] = v
			}
		}

		// All unrecognized files have `application/octet-stream`
		// So we continue our detection process.
		if ctype := metadata["Content-Type"]; ctype == "application/octet-stream" {
			// Continue probing content-type if its filesystem stream.
			if !mok {
				metadata["Content-Type"], err = probeContentType(reader)
				if err != nil {
					return nil, nil, err.Trace(alias, urlStr)
				}
			}
		}
	}
	return reader, metadata, nil
}

// putTargetRetention sets retention headers if any
func putTargetRetention(ctx context.Context, alias string, urlStr string, metadata map[string]string) *probe.Error {
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
func uploadSourceToTargetURL(ctx context.Context, urls URLs, progress io.Reader, encKeyDB map[string][]prefixSSEPair, preserve bool) URLs {
	sourceAlias := urls.SourceAlias
	sourceURL := urls.SourceContent.URL
	sourceVersion := urls.SourceContent.VersionID
	targetAlias := urls.TargetAlias
	targetURL := urls.TargetContent.URL
	length := urls.SourceContent.Size
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, urls.SourceContent.URL.Path))
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, urls.TargetContent.URL.Path))

	srcSSE := getSSE(sourcePath, encKeyDB[sourceAlias])
	tgtSSE := getSSE(targetPath, encKeyDB[targetAlias])

	var err *probe.Error
	var metadata = map[string]string{}
	var mode, until, legalHold string

	// add object retention fields in metadata for target, if target wants
	// to override defaults from source, usually happens in `cp` command.
	// for the most part source metadata is copied over.
	if urls.TargetContent.RetentionEnabled {
		m := minio.RetentionMode(strings.ToUpper(urls.TargetContent.RetentionMode))
		if !m.IsValid() {
			return urls.WithError(probe.NewError(errors.New("invalid retention mode")).Trace(targetURL.String()))
		}

		var dur uint64
		var unit minio.ValidityUnit
		dur, unit, err = parseRetentionValidity(urls.TargetContent.RetentionDuration)
		if err != nil {
			return urls.WithError(err.Trace(targetURL.String()))
		}

		mode = urls.TargetContent.RetentionMode

		until, err = getRetainUntilDate(dur, unit)
		if err != nil {
			return urls.WithError(err.Trace(sourceURL.String()))
		}
	}

	// add object legal hold fields in metadata for target, if target wants
	// to override defaults from source, usually happens in `cp` command.
	// for the most part source metadata is copied over.
	if urls.TargetContent.LegalHoldEnabled {
		switch minio.LegalHoldStatus(urls.TargetContent.LegalHold) {
		case minio.LegalHoldDisabled:
		case minio.LegalHoldEnabled:
		default:
			return urls.WithError(errInvalidArgument().Trace(urls.TargetContent.LegalHold))
		}
		legalHold = urls.TargetContent.LegalHold
	}

	for k, v := range urls.SourceContent.UserMetadata {
		metadata[http.CanonicalHeaderKey(k)] = v
	}
	for k, v := range urls.SourceContent.Metadata {
		metadata[http.CanonicalHeaderKey(k)] = v
	}

	// Optimize for server side copy if the host is same.
	if sourceAlias == targetAlias {
		// preserve new metadata and save existing ones.
		if preserve {
			currentMetadata, err := getAllMetadata(ctx, sourceAlias, sourceURL.String(), srcSSE, urls)
			if err != nil {
				return urls.WithError(err.Trace(sourceURL.String()))
			}
			for k, v := range currentMetadata {
				metadata[k] = v
			}
		}

		// Get metadata from target content as well
		for k, v := range urls.TargetContent.Metadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		// Get userMetadata from target content as well
		for k, v := range urls.TargetContent.UserMetadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		sourcePath := filepath.ToSlash(sourceURL.Path)
		if urls.SourceContent.RetentionEnabled {
			err = putTargetRetention(ctx, targetAlias, targetURL.String(), metadata)
			return urls.WithError(err.Trace(sourceURL.String()))
		}

		opts := CopyOptions{
			srcSSE:           srcSSE,
			tgtSSE:           tgtSSE,
			metadata:         filterMetadata(metadata),
			disableMultipart: urls.DisableMultipart,
			isPreserve:       preserve,
			storageClass:     urls.TargetContent.StorageClass,
		}

		err = copySourceToTargetURL(ctx, targetAlias, targetURL.String(), sourcePath, sourceVersion, mode, until,
			legalHold, length, progress, opts)
	} else {
		if urls.SourceContent.RetentionEnabled {
			// preserve new metadata and save existing ones.
			if preserve {
				currentMetadata, err := getAllMetadata(ctx, sourceAlias, sourceURL.String(), srcSSE, urls)
				if err != nil {
					return urls.WithError(err.Trace(sourceURL.String()))
				}
				for k, v := range currentMetadata {
					metadata[k] = v
				}
			}

			// Get metadata from target content as well
			for k, v := range urls.TargetContent.Metadata {
				metadata[http.CanonicalHeaderKey(k)] = v
			}

			// Get userMetadata from target content as well
			for k, v := range urls.TargetContent.UserMetadata {
				metadata[http.CanonicalHeaderKey(k)] = v
			}

			err = putTargetRetention(ctx, targetAlias, targetURL.String(), metadata)
			return urls.WithError(err.Trace(sourceURL.String()))
		}

		var reader io.ReadCloser
		// Proceed with regular stream copy.
		reader, metadata, err = getSourceStream(ctx, sourceAlias, sourceURL.String(), sourceVersion, true, srcSSE, preserve)
		if err != nil {
			return urls.WithError(err.Trace(sourceURL.String()))
		}
		defer reader.Close()

		// Get metadata from target content as well
		for k, v := range urls.TargetContent.Metadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		// Get userMetadata from target content as well
		for k, v := range urls.TargetContent.UserMetadata {
			metadata[http.CanonicalHeaderKey(k)] = v
		}

		putOpts := PutOptions{
			metadata:         filterMetadata(metadata),
			sse:              tgtSSE,
			storageClass:     urls.TargetContent.StorageClass,
			md5:              urls.MD5,
			disableMultipart: urls.DisableMultipart,
			isPreserve:       preserve,
		}

		if isReadAt(reader) {
			_, err = putTargetStream(ctx, targetAlias, targetURL.String(), mode, until,
				legalHold, reader, length, progress, putOpts)
		} else {
			_, err = putTargetStream(ctx, targetAlias, targetURL.String(), mode, until,
				legalHold, io.LimitReader(reader, length), length, progress, putOpts)
		}
	}
	if err != nil {
		return urls.WithError(err.Trace(sourceURL.String()))
	}

	return urls.WithError(nil)
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

	s3Config := NewS3Config(urlStr, hostCfg)

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
