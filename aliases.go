/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"regexp"
	"strings"

	"github.com/minio-io/minio/pkg/iodine"
)

// Alias for S3 hosts, saved in mc json configuration file
type mcAlias struct {
	Name string // Any alphanumeric string /^[a-zA-Z0-9-_]+$/
	URL  string // Eg.: https://s3.amazonaws.com/
}

// validAliasURL: use net/url.Parse to validate
var validAliasName = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$")

// Check if it is an aliased URL
func isValidAliasName(aliasName string) bool {
	return validAliasName.MatchString(aliasName)
}

// aliasExpand expands aliased (name:/path) to full URL
func aliasExpand(aliasedURL string) (newURL string, err error) {
	url, err := url.Parse(aliasedURL)
	if err != nil {
		// Not a valid URL. Return error
		return aliasedURL, iodine.New(err, nil)
	}

	// Not an aliased URL
	if url.Scheme == "" {
		return aliasedURL, nil
	}

	// load from json config file
	config, err := getMcConfig()
	if err != nil {
		return "", iodine.New(err, nil)
	}

	for aliasName, expandedURL := range config.Aliases {
		if strings.HasPrefix(aliasedURL, aliasName) {
			// Match found. Expand it.
			return strings.Replace(aliasedURL, aliasName+":", expandedURL, 1), nil
		}
	}

	// No matching alias. Return the original
	return aliasedURL, nil
}
