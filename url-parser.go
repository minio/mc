/*
 * Modern Copy, (C) 2014, 2015 Minio, Inc.
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
	"path"
	"path/filepath"
	"strings"

	"net/url"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

// TODO - y4m4 - this file needs to be cleaned up before we make a release
//
// The actual work on this file is about how do we handle generic
// conversions for both file:/// and http:// in a simple way.
//
// I propose we write this as a proper struct and its methods.

// URLType defines supported storage protocols
type urlType int

const (
	urlUnknown       urlType = iota // Unknown type
	urlObjectStorage                // Minio and S3 compatible object storage
	urlFile                         // POSIX compatible file systems
)

// urlType detects the type of URL
func getURLType(urlStr string) (uType urlType, err error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlUnknown, iodine.New(err, nil)
	}
	switch u.Scheme {
	case "http":
		fallthrough
	case "https":
		return urlObjectStorage, nil
	case "file":
		fallthrough
	case "":
		return urlFile, nil
	default:
		return urlUnknown, nil
	}
}

// isValidURL checks the validity of supported URL types
func isValidURL(urlStr string) bool {
	u, e := getURLType(urlStr)
	if e != nil || u == urlUnknown {
		return false
	}
	return true
}

// isValidURL checks the validity of supported URL types
func isValidFileURL(urlStr string) bool {
	utype, e := getURLType(urlStr)
	if e != nil || utype != urlFile {
		return false
	}
	return true
}

// fixFileURL rewrites file URL to proper file:///path/to/ form.
func fixFileURL(urlStr string) (fixedURL string, err error) {
	if urlStr == "" {
		return "", iodine.New(errEmptyURL, nil)
	}
	utype, e := getURLType(urlStr)
	if e != nil || utype != urlFile {
		return "", iodine.New(e, nil)
	}

	u, e := url.Parse(urlStr)
	if e != nil {
		return "", iodine.New(e, nil)
	}

	// file:///path should always have empty host
	if u.Host != "" {
		// Not really a file URL. Host is not empty.
		return "", iodine.New(errInvalidURL, nil)
	}
	// do not use u.Scheme since that would construct a path in the form
	// file:// which is an invalid file but url Parse doesn't report error
	// so we construct manually instead as an absolute path
	path, _ := filepath.Abs(u.Path)
	fixedURL = "file://" + path
	return fixedURL, nil

}

// url2Object converts URL to bucket and objectname
func url2Object(urlStr string) (bucketName, objectName string, err error) {
	u, err := url.Parse(urlStr)
	if u.Path == "" {
		// No bucket name passed. It is a valid case
		return "", "", nil
	}
	// url is of scheme file, behave differently by returning
	// directory and file instead using filepath.Split function
	if u.Scheme == "file" {
		bucketName, objectName = filepath.Split(u.Path)
		return bucketName, objectName, nil
	}
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
	return bucketName, objectName, nil
}

// url2Bucket converts URL to bucket name
func url2Bucket(urlStr string) (bucketName string, err error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	if u.Scheme == "file" {
		bucketName, objectName, err := url2Object(urlStr)
		if err != nil {
			return "", iodine.New(err, nil)
		}
		return path.Join(bucketName, objectName), nil
	}
	bucketName, objectName, err := url2Object(urlStr)
	if objectName != "" {
		// objectName also provided invalid argument
		return "", iodine.New(client.InvalidArgument{}, nil)
	}
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return bucketName, nil
}

// parseURL extracts URL string from a single cmd-line argument
func parseURL(arg string, aliases map[string]string) (urlStr string, err error) {
	// Check and expand Alias
	urlStr, err = aliasExpand(arg, aliases)
	if err != nil {
		return "", iodine.New(err, nil)
	}

	if !isValidURL(urlStr) {
		return "", iodine.New(errUnsupportedScheme, nil)
	}
	// If it is a file URL, rewrite to file:///path/to form
	if isValidFileURL(urlStr) {
		return fixFileURL(urlStr)
	}

	return urlStr, nil
}

// parseURL extracts multiple URL strings from a single cmd-line argument
func parseURLs(c *cli.Context) (urlStr []string, err error) {
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to get config")
	}
	for _, arg := range c.Args() {
		u, err := parseURL(arg, config.Aliases)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		urlStr = append(urlStr, u)
	}
	return urlStr, iodine.New(err, nil)
}
