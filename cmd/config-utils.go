/*
 * MinIO Client (C) 2015, 2016 MinIO, Inc.
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

package cmd

import "strings"

var validAPIs = []string{"S3v4", "S3v2"}

const (
	accessKeyMinLen = 3
	secretKeyMinLen = 8
)

// isValidAccessKey - validate access key for right length.
func isValidAccessKey(accessKey string) bool {
	if accessKey == "" {
		return true
	}
	return len(accessKey) >= accessKeyMinLen
}

// isValidSecretKey - validate secret key for right length.
func isValidSecretKey(secretKey string) bool {
	if secretKey == "" {
		return true
	}
	return len(secretKey) >= secretKeyMinLen
}

// trimTrailingSeparator - Remove trailing separator.
func trimTrailingSeparator(hostURL string) string {
	separator := string(newClientURL(hostURL).Separator)
	return strings.TrimSuffix(hostURL, separator)
}

// isValidHostURL - validate input host url.
func isValidHostURL(hostURL string) (ok bool) {
	if strings.TrimSpace(hostURL) != "" {
		url := newClientURL(hostURL)
		if url.Scheme == "https" || url.Scheme == "http" {
			if url.Path == "/" {
				ok = true
			}
		}
	}
	return ok
}

// isValidAPI - Validates if API signature string of supported type.
func isValidAPI(api string) (ok bool) {
	switch strings.ToLower(api) {
	case "s3v2", "s3v4":
		ok = true
	}
	return ok
}

// isValidLookup - validates if bucket lookup is of valid type
func isValidLookup(lookup string) (ok bool) {
	l := strings.ToLower(strings.TrimSpace(lookup))
	for _, v := range []string{"dns", "path", "auto"} {
		if l == v {
			return true
		}
	}
	return false
}

// isValidPath - validates the alias path config
func isValidPath(path string) (ok bool) {
	l := strings.ToLower(strings.TrimSpace(path))
	for _, v := range []string{"on", "off", "auto"} {
		if l == v {
			return true
		}
	}
	return false
}
