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
	"os"
	"regexp"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
)

// isValidAliasName - Check if aliasName is a valid alias.
func isValidAliasName(aliasName string) bool {
	return regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$").MatchString(aliasName)
}

// normalizeAliasedURL - remove any preceding separators.
func normalizeAliasedURL(aliasedURL string) string {
	aliasedURL = strings.TrimPrefix(aliasedURL, string(os.PathSeparator))
	return aliasedURL
}

// getAliasURL expands aliased (name/path) to full URL, used by url-parser.
func getAliasURL(aliasedURL string) (string, *probe.Error) {
	config, err := loadMcConfig()
	if err != nil {
		return aliasedURL, err.Trace(aliasedURL)
	}
	if strings.HasPrefix(aliasedURL, "https") || strings.HasPrefix(aliasedURL, "http") {
		return aliasedURL, nil
	}
	for hostURL := range config.Hosts {
		url := client.NewURL(hostURL)
		if strings.Contains(aliasedURL, url.Host) {
			return url.Scheme + url.SchemeSeparator + aliasedURL, nil
		}
	}
	normalizedAliasURL := normalizeAliasedURL(aliasedURL)
	for aliasName, aliasValue := range config.Aliases {
		if strings.HasPrefix(normalizedAliasURL, aliasName) {
			// Match found. Expand it.
			splits := strings.SplitN(normalizedAliasURL, aliasName, 2)
			if len(splits) == 1 {
				return aliasedURL, nil // Not an aliased URL. Return as is.
			}
			if len(splits[0]) == 0 && len(splits[1]) == 0 {
				return aliasValue, nil // exact match.
			}
			// Matched, but path needs to be joined.
			return urlJoinPath(aliasValue, splits[1]), nil
		}
	}
	return aliasedURL, nil // No matching alias found. Return as is.
}
