// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
