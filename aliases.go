/*
 * Mini Object Storage, (C) 2015 Minio, Inc.
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
	"strings"
)

// Alias for S3 hosts, saved in mc json configuration file
type mcAlias struct {
	Name string // Any alphanumeric string [a-zA-Z_][0-9a-zA-Z_]
	URL  string // Eg.: https://s3.amazonaws.com/
}

// aliasExpand expands aliased (name:/path) URL to normal (http(s)://host:port/path) URL
func aliasExpand(URL string) (newURL string, err error) {
	if strings.HasPrefix(URL, "http") || strings.HasPrefix(URL, "https") {
		//Not an alias. Return the original URL
		return URL, nil
	}
	config := getMcConfig()

	for _, alias := range config.Aliases {
		if strings.HasPrefix(URL, alias.Name) {
			newURL = strings.Replace(URL, alias.Name+":", alias.URL, 1)
			break
		}
	}
	if newURL == "" {
		return URL, fmt.Errorf("No matching alias for URL [%s]", URL)
	}
	return newURL, nil
}
