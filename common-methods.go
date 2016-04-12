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

package main

import (
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Check if the target URL represents folder. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	targetURLParse := newClientURL(targetURL)
	_, targetContent, err := url2Stat(targetURL)
	if err != nil {
		_, aliasedTargetURL, _ := mustExpandAlias(targetURL)
		if aliasedTargetURL == targetURL {
			return false
		}
		if targetURLParse.Path == string(targetURLParse.Separator) && targetURLParse.Scheme != "" {
			return false
		}
		if strings.HasSuffix(targetURLParse.Path, string(targetURLParse.Separator)) {
			return true
		}
		return false
	}
	if !targetContent.Type.IsDir() { // Target is a dir.
		return false
	}
	return true
}

// getSource gets a reader from URL.
func getSourceStream(urlStr string) (reader io.Reader, err *probe.Error) {
	alias, urlStrFull, _, err := expandAlias(urlStr)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	return getSourceStreamFromAlias(alias, urlStrFull)
}

// getSourceStreamFromAlias gets a reader from URL.
func getSourceStreamFromAlias(alias string, urlStr string) (reader io.Reader, err *probe.Error) {
	sourceClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return nil, err.Trace(alias, urlStr)
	}
	reader, err = sourceClnt.Get()
	if err != nil {
		return nil, err.Trace(alias, urlStr)
	}
	return reader, nil
}

// putTargetStreamFromAlias writes to URL from Reader.
func putTargetStreamFromAlias(alias string, urlStr string, reader io.Reader, size int64, progress io.Reader) (int64, *probe.Error) {
	targetClnt, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return 0, err.Trace(alias, urlStr)
	}
	contentType := guessURLContentType(urlStr)
	var n int64
	n, err = targetClnt.Put(reader, size, contentType, progress)
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
	return putTargetStreamFromAlias(alias, urlStrFull, reader, size, nil)
}

// copyTargetStreamFromAlias copies to URL from source.
func copySourceStreamFromAlias(alias string, urlStr string, source string, size int64, progress io.Reader) *probe.Error {
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

// newClientFromAlias gives a new client interface for matching
// alias entry in the mc config file. If no matching host config entry
// is found, fs client is returned.
func newClientFromAlias(alias string, urlStr string) (Client, *probe.Error) {
	hostCfg := mustGetHostConfig(alias)
	if hostCfg == nil {
		// No matching host config. So we treat it like a
		// filesystem.
		fsClient, err := fsNew(urlStr)
		if err != nil {
			return nil, err.Trace(alias, urlStr)
		}
		return fsClient, nil
	}

	// We have a valid alias and hostConfig. We populate the
	// credentials from the match found in the config file.
	s3Config := new(Config)

	// secretKey retrieved from the environement overrides the one
	// present in the config file
	keysPairEnv := os.Getenv("MC_SECRET_" + alias)
	keysPairArray := strings.Split(keysPairEnv, ":")
	var accessKeyEnv, secretKeyEnv string
	if len(keysPairArray) >= 1 {
		accessKeyEnv = keysPairArray[0]
	}
	if len(keysPairArray) >= 2 {
		secretKeyEnv = keysPairArray[1]
	}
	if len(keysPairEnv) > 0 &&
		isValidAccessKey(accessKeyEnv) && isValidSecretKey(secretKeyEnv) {
		s3Config.AccessKey = accessKeyEnv
		s3Config.SecretKey = secretKeyEnv
	} else {
		if len(keysPairEnv) > 0 {
			console.Errorln("Access/Secret keys associated to `" + alias + "' " +
				"are found in your environment but not suitable for use. " +
				"Falling back to the standard config.")
		}
		s3Config.AccessKey = hostCfg.AccessKey
		s3Config.SecretKey = hostCfg.SecretKey
	}

	s3Config.Signature = hostCfg.API
	s3Config.AppName = "mc"
	s3Config.AppVersion = mcVersion
	s3Config.AppComments = []string{os.Args[0], runtime.GOOS, runtime.GOARCH}
	s3Config.HostURL = urlStr
	s3Config.Debug = globalDebug

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
