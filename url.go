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
	"net/url"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// getURL extracts URL string from a single cmd-line argument
func getURL(arg string, aliases map[string]string) (urlStr string, err error) {
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
	if client.GetURLType(urlStr) == client.URLUnknown {
		return "", iodine.New(errUnsupportedScheme{scheme: client.URLUnknown}, map[string]string{"URL": urlStr})
	}
	return urlStr, nil
}

// getURLs extracts multiple URL strings from a single cmd-line argument
func getURLs(args []string, aliases map[string]string) (urls []string, err error) {
	for _, arg := range args {
		u, err := getURL(arg, aliases)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		urls = append(urls, u)
	}
	return urls, nil
}
