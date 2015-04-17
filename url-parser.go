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
	"regexp"
	"runtime"
	"strings"

	"net/url"
	"path/filepath"

	"github.com/minio-io/cli"
	"github.com/minio-io/minio/pkg/iodine"
)

type urlType int

const (
	urlUnknown urlType = iota // Unknown type
	urlS3                     // Minio and S3 compatible object storage
	urlFS                     // POSIX compatible file systems
)

type parsedURL struct {
	url        *url.URL
	scheme     urlType
	bucketName string
	objectName string
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

	if u.Scheme == "file" {
		return urlFS
	}

	// MS Windows OS: Match drive letters
	if runtime.GOOS == "windows" {
		if regexp.MustCompile(`^[a-zA-Z]?$`).MatchString(u.Scheme) {
			return urlFS
		}
	}

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

func newURL(urlStr string) (*parsedURL, error) {
	bucketName, objectName, err := url2Object(urlStr)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	parsedURL := &parsedURL{
		url:        u,
		scheme:     getURLType(urlStr),
		bucketName: bucketName,
		objectName: objectName,
	}
	return parsedURL, nil
}

func (u *parsedURL) String() string {
	switch u.scheme {
	case urlFS:
		var p string
		switch runtime.GOOS {
		case "windows":
			p, _ = filepath.Abs(u.url.String())
			return p
		default:
			p, _ = filepath.Abs(u.url.Path)
			fileURL := "file://" + p
			return fileURL
		}
	}
	return u.url.String()
}

// parseURL extracts URL string from a single cmd-line argument
func parseURL(arg string, aliases map[string]string) (url *parsedURL, err error) {
	// Check and expand Alias
	urlStr, err := aliasExpand(arg, aliases)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	u, err := newURL(urlStr)
	if u.scheme == urlUnknown {
		return nil, iodine.New(errUnsupportedScheme{scheme: urlUnknown}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return u, nil
}

// parseURL extracts multiple URL strings from a single cmd-line argument
func parseURLs(c *cli.Context) (urlParsers []*parsedURL, err error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	for _, arg := range c.Args() {
		u, err := parseURL(arg, config.Aliases)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		urlParsers = append(urlParsers, u)
	}
	return urlParsers, nil
}
