/*
 * Modern Copy, (C) 2015 Minio, Inc.
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
	"strings"

	"net/url"
	"path/filepath"

	"github.com/minio-io/cli"
	"github.com/minio-io/minio/pkg/iodine"
)

type urlType int

const (
	urlUnknown       urlType = iota // Unknown type
	urlObjectStorage                // Minio and S3 compatible object storage
	urlFile                         // POSIX compatible file systems
)

type urlParser struct {
	url        *url.URL
	urlType    urlType
	bucketName string
	objectName string
}

func getURLType(scheme string) urlType {
	switch scheme {
	case "http":
		fallthrough
	case "https":
		return urlObjectStorage
	case "file":
		fallthrough
	case "":
		return urlFile
	default:
		return urlUnknown
	}
}

// url2Object converts URL to bucket and objectname
func url2Object(u *url.URL) (bucketName, objectName string) {
	// if url is of scheme file, behave differently by returning
	// directory and file instead
	switch u.Scheme {
	case "file":
		fallthrough
	case "":
		bucketName, objectName = filepath.Split(u.Path)
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
	return bucketName, objectName
}

func newURL(urlStr string) (*urlParser, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	bucketName, objectName := url2Object(u)
	parsedURL := &urlParser{
		url:        u,
		urlType:    getURLType(u.Scheme),
		bucketName: bucketName,
		objectName: objectName,
	}
	return parsedURL, nil
}

func (u *urlParser) String() string {
	switch u.urlType {
	case urlFile:
		p, _ := filepath.Abs(u.url.Path)
		fixedURL := "file://" + p
		return fixedURL
	}
	return u.url.String()
}

// parseURL extracts URL string from a single cmd-line argument
func parseURL(arg string, aliases map[string]string) (url *urlParser, err error) {
	// Check and expand Alias
	urlStr, err := aliasExpand(arg, aliases)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	u, err := newURL(urlStr)
	if u.urlType == urlUnknown {
		return nil, iodine.New(errUnsupportedScheme{scheme: urlUnknown}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return u, nil
}

// parseURL extracts multiple URL strings from a single cmd-line argument
func parseURLs(c *cli.Context) (urlParsers []*urlParser, err error) {
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
