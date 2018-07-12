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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"golang.org/x/net/lex/httplex"
)

// parse and return encryption key pairs per alias.
func getEncKeys(ctx *cli.Context) (map[string][]prefixSSEPair, *probe.Error) {
	sseKeys := os.Getenv("MC_ENCRYPT_KEY")
	if key := ctx.String("encrypt-key"); key != "" {
		sseKeys = key
	}

	encKeyDB, err := parseAndValidateEncryptionKeys(sseKeys)
	if err != nil {
		return nil, err.Trace(sseKeys)
	}
	return encKeyDB, nil
}

// Check if the passed URL represents a folder. It may or may not exist yet.
// If it exists, we can easily check if it is a folder, if it doesn't exist,
// we can guess if the url is a folder from how it looks.
func isAliasURLDir(aliasURL string, keys map[string][]prefixSSEPair) bool {
	// If the target url exists, check if it is a directory
	// and return immediately.
	_, targetContent, err := url2Stat(aliasURL, false, keys)
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

// getSourceStreamFromURL gets a reader from URL.
func getSourceStreamFromURL(urlStr string, encKeyDB map[string][]prefixSSEPair) (reader io.Reader, err *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	sseKey := getSSEKey(urlStr, encKeyDB[alias])
	reader, _, err = getSourceStream(alias, urlStrFull, false, sseKey)
	return reader, err
}

// getSourceStream gets a reader from URL.
func getSourceStream(alias string, urlStr string, fetchStat bool, sseKey string) (reader io.Reader, metadata map[string]string, err *probe.Error) {
	sourceClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}
	reader, err = sourceClnt.Get(sseKey)
	if err != nil {
		return nil, nil, err.Trace(alias, urlStr)
	}
	metadata = map[string]string{}
	if fetchStat {
		st, err := sourceClnt.Stat(false, true, sseKey)
		if err != nil {
			return nil, nil, err.Trace(alias, urlStr)
		}
		for k, v := range st.Metadata {
			if httplex.ValidHeaderFieldName(k) &&
				httplex.ValidHeaderFieldValue(v) {
				metadata[k] = v
			}
		}
	}
	return reader, metadata, nil
}

// putTargetStream writes to URL from Reader.
func putTargetStream(ctx context.Context, alias string, urlStr string, reader io.Reader, size int64, metadata map[string]string, progress io.Reader, sseKey string) (int64, *probe.Error) {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}
	n, err := targetClnt.Put(ctx, reader, size, metadata, progress, sseKey)
	if err != nil {
		return n, err.Trace(alias, urlStr)
	}
	return n, nil
}

// putTargetStreamWithURL writes to URL from reader. If length=-1, read until EOF.
func putTargetStreamWithURL(urlStr string, reader io.Reader, size int64, sseKey string) (int64, *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}
	contentType := guessURLContentType(urlStr)
	metadata := map[string]string{
		"Content-Type": contentType,
	}
	return putTargetStream(context.Background(), alias, urlStrFull, reader, size, metadata, nil, sseKey)
}

// copySourceToTargetURL copies to targetURL from source.
func copySourceToTargetURL(alias string, urlStr string, source string, size int64, progress io.Reader, srcSSEKey, tgtSSEKey string) *probe.Error {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return err.Trace(alias, urlStr)
	}
	err = targetClnt.Copy(source, size, progress, srcSSEKey, tgtSSEKey)
	if err != nil {
		return err.Trace(alias, urlStr)
	}
	return nil
}

// uploadSourceToTargetURL - uploads to targetURL from source.
// optionally optimizes copy for object sizes <= 5GiB by using
// server side copy operation.
func uploadSourceToTargetURL(ctx context.Context, urls URLs, progress io.Reader) URLs {
	sourceAlias := urls.SourceAlias
	sourceURL := urls.SourceContent.URL
	targetAlias := urls.TargetAlias
	targetURL := urls.TargetContent.URL
	length := urls.SourceContent.Size
	// Optimize for server side copy if the host is same.
	if sourceAlias == targetAlias {
		sourcePath := filepath.ToSlash(sourceURL.Path)
		err := copySourceToTargetURL(targetAlias, targetURL.String(), sourcePath, length, progress, urls.SrcSSEKey, urls.TgtSSEKey)
		if err != nil {
			return urls.WithError(err.Trace(sourceURL.String()))
		}
	} else {
		// Proceed with regular stream copy.
		reader, metadata, err := getSourceStream(sourceAlias, sourceURL.String(), true, urls.SrcSSEKey)
		if err != nil {
			return urls.WithError(err.Trace(sourceURL.String()))
		}
		// Get metadata from target content as well
		if urls.TargetContent.Metadata != nil {
			for k, v := range urls.TargetContent.Metadata {
				metadata[k] = v
			}
		}
		if urls.SrcSSEKey != "" {
			delete(metadata, "X-Amz-Server-Side-Encryption-Customer-Algorithm")
			delete(metadata, "X-Amz-Server-Side-Encryption-Customer-Key-Md5")
		}

		_, err = putTargetStream(ctx, targetAlias, targetURL.String(), reader, length, metadata, progress, urls.TgtSSEKey)
		if err != nil {
			return urls.WithError(err.Trace(targetURL.String()))
		}
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

	s3Config := newS3Config(urlStr, hostCfg)

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
