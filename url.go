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
	"strings"

	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/pkg/client"
)

// ``...`` recursiveSeparator
const (
	recursiveSeparator = "..."
)

// isURLRecursive - find out if requested url is recursive
func isURLRecursive(urlStr string) bool {
	return strings.HasSuffix(urlStr, recursiveSeparator)
}

// stripRecursiveURL - Strip "..." from the URL if present.
func stripRecursiveURL(urlStr string) string {
	if !isURLRecursive(urlStr) {
		return urlStr
	}
	urlStr = strings.TrimSuffix(urlStr, recursiveSeparator)
	if urlStr == "" {
		urlStr = "."
	}
	return urlStr
}

// getExpandedURL - extracts URL string from a single cmd-line argument
func getExpandedURL(arg string, aliases map[string]string) (urlStr string, err *probe.Error) {
	// Check and expand Alias
	urlStr, err = aliasExpand(arg, aliases)
	if err != nil {
		return "", err.Trace()
	}
	if _, err := client.Parse(urlStr); err != nil {
		// Not a valid URL. Return error
		return "", probe.NewError(errInvalidURL{urlStr})
	}
	return urlStr, nil
}

// getExpandedURLs - extracts multiple URL strings from a single cmd-line argument
func getExpandedURLs(args []string, aliases map[string]string) (urls []string, err *probe.Error) {
	for _, arg := range args {
		u, err := getExpandedURL(arg, aliases)
		if err != nil {
			return nil, err.Trace()
		}
		urls = append(urls, u)
	}
	return urls, nil
}

// args2URLs extracts source and target URLs from command-line args.
func args2URLs(args []string) ([]string, *probe.Error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, err.Trace()

	}
	// Convert arguments to URLs: expand alias, fix format...
	URLs, err := getExpandedURLs(args, config.Aliases)
	if err != nil {
		return nil, err.Trace()
	}
	return URLs, nil
}
