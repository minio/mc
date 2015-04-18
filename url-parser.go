/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"net/url"
	"path/filepath"

	"github.com/minio-io/minio/pkg/iodine"
)

type urlType int

const (
	urlUnknown urlType = iota // Unknown type
	urlS3                     // Minio and S3 compatible object storage
	urlFS                     // POSIX compatible file systems
)

// guessPossibleURL - provide guesses for possible mistakes in user input url
func guessPossibleURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	if u.Scheme == "file" || !strings.Contains(urlStr, ":///") {
		possibleURL := u.Scheme + ":///" + u.Host + u.Path
		guess := fmt.Sprintf("Did you mean? %s", possibleURL)
		return guess
	}
	// TODO(y4m4) - add more guesses if possible
	return ""
}

// getURLType returns the type of URL.
func getURLType(urlStr string) urlType {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlUnknown
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return urlS3
	}

	// while Scheme file, host should be empty
	if u.Scheme == "file" && u.Host == "" && strings.Contains(urlStr, ":///") {
		return urlFS
	}

	// MS Windows OS: Match drive letters
	if runtime.GOOS == "windows" {
		if regexp.MustCompile(`^[a-zA-Z]?$`).MatchString(u.Scheme) {
			return urlFS
		}
	}

	// local path, without the file:///
	if u.Scheme == "" {
		return urlFS
	}

	return urlUnknown
}

// url2Object converts URL to bucket and objectname
func url2Object(urlStr string) (bucketName, objectName string, err error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", iodine.New(err, nil)
	}

	switch getURLType(urlStr) {
	case urlFS:
		if runtime.GOOS == "windows" {
			bucketName, objectName = filepath.Split(u.String())
		} else {
			bucketName, objectName = filepath.Split(u.Path)
		}
	default:
		splits := strings.SplitN(u.Path, "/", 3)
		switch len(splits) {
		case 0, 1:
			bucketName = ""
			objectName = ""
		case 2:
			bucketName = splits[1]
			objectName = ""
		case 3:
			bucketName = splits[1]
			objectName = splits[2]
		}
	}
	return bucketName, objectName, nil
}

// parseURL extracts URL string from a single cmd-line argument
func parseURL(arg string, aliases map[string]string) (urlStr string, err error) {
	_, err = url.Parse(arg)
	if err != nil {
		// Not a valid URL. Return error
		return "", iodine.New(errInvalidURL{arg}, nil)
	}
	// Check and expand Alias
	urlStr, err = aliasExpand(arg, aliases)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	if getURLType(urlStr) == urlUnknown {
		return "", iodine.New(errUnsupportedScheme{scheme: urlUnknown}, map[string]string{"URL": urlStr})
	}
	return urlStr, nil
}

// parseURL extracts multiple URL strings from a single cmd-line argument
func parseURLs(args []string, aliases map[string]string) (urls []string, err error) {
	for _, arg := range args {
		u, err := parseURL(arg, aliases)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		urls = append(urls, u)
	}
	return urls, nil
}
