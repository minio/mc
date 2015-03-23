/*
 * Minimalist Object Storage, (C) 2014, 2015 Minio, Inc.
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

	"github.com/codegangsta/cli"
)

// url2Object converts URL to bucket and objectname
func url2Object(urlStr string) (bucketName, objectName string, err error) {
	url, err := url.Parse(urlStr)
	if url.Path == "" {
		// No bucket name passed. It is a valid case.
		return "", "", nil
	}
	splits := strings.SplitN(url.Path, "/", 3)
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
	bucketName, _, err = url2Object(urlStr)
	return bucketName, err
}

// parseURL extracts URL string from a single cmd-line argument
func parseURL(c *cli.Context) (urlStr string, err error) {
	urlStr = c.Args().First()
	// Use default host if no argument is passed
	if urlStr == "" {
		// Load config file
		config, err := getMcConfig()
		if err != nil {
			return "", err
		}
		urlStr = config.DefaultHost
	}
	// Check and expand Alias
	urlStr, err = aliasExpand(urlStr)

	if err != nil {
		return "", err
	}

	return urlStr, err
}
