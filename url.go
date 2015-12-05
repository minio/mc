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
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio/pkg/contentdb"
)

func isURLVirtualHostStyle(hostURL string) bool {
	matchS3, _ := filepath.Match("*.s3*.amazonaws.com", hostURL)
	matchGoogle, _ := filepath.Match("*.storage.googleapis.com", hostURL)
	return matchS3 || matchGoogle
}

// urlJoinPath Join a path to existing URL.
func urlJoinPath(url1, url2 string) string {
	u1 := client.NewURL(url1)
	u2 := client.NewURL(url2)
	return client.JoinURLs(u1, u2).String()
}

// args2URLs extracts source and target URLs from command-line args.
func args2URLs(args []string) ([]string, *probe.Error) {
	// convert arguments to URLs: expand alias, fix format...
	URLs := []string{}
	for _, arg := range args {
		aliasedURL, err := getAliasURL(arg)
		if err != nil {
			return nil, err.Trace(arg)
		}
		URLs = append(URLs, aliasedURL)
	}
	return URLs, nil
}

// url2Client convenience wrapper for getNewClient.
func url2Client(urlStr string) (client.Client, *probe.Error) {
	urlConfig, err := getHostConfig(urlStr)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	client, err := getNewClient(urlStr, urlConfig)
	if err != nil {
		return nil, err.Trace(urlStr)
	}
	return client, nil
}

// url2Stat returns stat info for URL.
func url2Stat(urlStr string) (client client.Client, content *client.Content, err *probe.Error) {
	client, err = url2Client(urlStr)
	if err != nil {
		return nil, nil, err.Trace(urlStr)
	}
	content, err = client.Stat()
	if err != nil {
		return nil, nil, err.Trace(urlStr)
	}
	return client, content, nil
}

// isURLPrefixExists - check if object key prefix exists.
func isURLPrefixExists(urlPrefix string, incomplete bool) bool {
	clnt, err := url2Client(urlPrefix)
	if err != nil {
		return false
	}
	isRecursive := true
	isIncomplete := incomplete
	for entry := range clnt.List(isRecursive, isIncomplete) {
		if entry.Err != nil {
			return false
		}
		return true
	}
	return false
}

// guessURLContentType - guess content-type of the URL.
// on failure just return 'application/octet-stream'.
func guessURLContentType(urlStr string) string {
	url := client.NewURL(urlStr)
	extension := strings.TrimPrefix(filepath.Ext(url.Path), ".")
	contentType, e := contentdb.Lookup(extension)
	if e != nil {
		return "application/octet-stream"
	}
	return contentType
}
