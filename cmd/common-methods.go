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
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/minio/minio/pkg/probe"
)

// Check if the target URL represents folder. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	_, targetContent, err := url2Stat(targetURL)
	if err == nil {
		return targetContent.Type.IsDir()
	}
	_, aliasedTargetURL, _ := mustExpandAlias(targetURL)
	if aliasedTargetURL == targetURL {
		return false
	}
	// targetURL is an aliased path, continue guessing if the path
	// is meant to point to a bucket or directory
	pathURL := filepath.FromSlash(targetURL)
	fields := strings.Split(pathURL, string(filepath.Separator))
	switch len(fields) {
	case 0, 1:
		return false
	case 2:
		return true
	default:
		return strings.HasSuffix(pathURL, string(filepath.Separator))
	}
}

// getSource gets a reader from URL.
func getSourceStream(urlStr string) (reader io.Reader, err *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	reader, _, err = getSourceStreamFromAlias(alias, urlStrFull)
	return reader, err
}

// getSourceStreamFromAlias gets a reader from URL.
func getSourceStreamFromAlias(alias string, urlStr string) (reader io.Reader, metadata map[string][]string, err *probe.Error) {
	sourceClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}
	reader, metadata, err = sourceClnt.Get()
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}
	return reader, metadata, nil
}

// putTargetStreamFromAlias writes to URL from Reader.
func putTargetStreamFromAlias(alias string, urlStr string, reader io.Reader, size int64, metadata map[string][]string, progress io.Reader) (int64, *probe.Error) {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}
	n, err := targetClnt.Put(reader, size, metadata, progress)
	if err != nil {
		return n, err.Trace(alias, urlStr)
	}
	return n, nil
}

// putTargetStream writes to URL from reader. If length=-1, read until EOF.
func putTargetStream(urlStr string, reader io.Reader, size int64) (int64, *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}
	contentType := guessURLContentType(urlStr)
	metadata := map[string][]string{
		"Content-Type": {contentType},
	}
	return putTargetStreamFromAlias(alias, urlStrFull, reader, size, metadata, nil)
}

// copySourceToTargetURL copies to targetURL from source.
func copySourceToTargetURL(alias string, urlStr string, source string, size int64, progress io.Reader) *probe.Error {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return err.Trace(alias, urlStr)
	}
	err = targetClnt.Copy(source, size, progress)
	if err != nil {
		return err.Trace(alias, urlStr)
	}
	return nil
}

// uploadSourceToTargetURL - uploads to targetURL from source.
// optionally optimizes copy for object sizes <= 5GiB by using
// server side copy operation.
func uploadSourceToTargetURL(urls URLs, progress io.Reader) URLs {
	sourceAlias := urls.SourceAlias
	sourceURL := urls.SourceContent.URL
	targetAlias := urls.TargetAlias
	targetURL := urls.TargetContent.URL
	length := urls.SourceContent.Size

	// Optimize for server side copy if object is <= 5GiB and the host is same.
	if length <= globalMaximumPutSize && sourceAlias == targetAlias {
		sourcePath := filepath.ToSlash(sourceURL.Path)
		err := copySourceToTargetURL(targetAlias, targetURL.String(), sourcePath, length, progress)
		if err != nil {
			return urls.WithError(err.Trace(sourceURL.String()))
		}
	} else {
		// Proceed with regular stream copy.
		reader, metadata, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
		if err != nil {
			return urls.WithError(err.Trace(sourceURL.String()))
		}
		_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, metadata, progress)
		if err != nil {
			return urls.WithError(err.Trace(targetURL.String()))
		}
	}
	return urls.WithError(nil)
}

// newClientFromAlias gives a new client interface for matching
// alias entry in the mc config file. If no matching host config entry
// is found, fs client is returned.
func newClientFromAlias(alias string, urlStr string) (Client, *probe.Error) {
	s3Config, err := buildS3Config(alias, urlStr)
	if err != nil {
		// No matching host config. So we treat it like a
		// filesystem.
		fsClient, fsErr := fsNew(urlStr)
		if fsErr != nil {
			return nil, fsErr.Trace(alias, urlStr)
		}
		return fsClient, nil
	}

	s3Client, err := s3New(s3Config)
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
