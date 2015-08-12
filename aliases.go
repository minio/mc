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
	"regexp"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

// validAliasURL: use net/url.Parse to validate
var validAliasName = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$")

func isAliasReserved(aliasName string) bool {
	// help is reserved argument
	switch aliasName {
	case "help":
		fallthrough
	case "private":
		fallthrough
	case "readonly":
		fallthrough
	case "public":
		fallthrough
	case "authenticated":
		return true
	default:
		return false
	}
}

// Check if it is an aliased URL
func isValidAliasName(aliasName string) bool {
	return validAliasName.MatchString(aliasName)
}

// aliasExpand expands aliased (name:/path) to full URL, used by url-parser
func aliasExpand(aliasedURL string, aliases map[string]string) (string, *probe.Error) {
	u, err := client.Parse(aliasedURL)
	if err != nil {
		return aliasedURL, probe.NewError(errInvalidURL{URL: aliasedURL})
	}
	// proper URL
	if u.Host != "" {
		return aliasedURL, nil
	}
	for aliasName, expandedURL := range aliases {
		if strings.HasPrefix(aliasedURL, aliasName+":") {
			// Match found. Expand it.
			splits := strings.Split(aliasedURL, ":")
			// if expandedURL is missing, return aliasedURL treat it like fs
			if expandedURL == "" {
				return aliasedURL, nil
			}
			// if more splits found return
			if len(splits) == 2 {
				// remove any prefixed slashes
				trimmedURL := expandedURL + "/" + strings.TrimPrefix(strings.TrimPrefix(splits[1], "/"), "\\")
				u, err := client.Parse(trimmedURL)
				if err != nil {
					return aliasedURL, probe.NewError(errInvalidURL{URL: aliasedURL})
				}
				return u.String(), nil
			}
			return aliasedURL, nil
		}
	}
	return aliasedURL, nil
}
