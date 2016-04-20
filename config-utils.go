/*
 * Minio Client (C) 2015, 2016 Minio, Inc.
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
)

var validAPIs = []string{"S3v4", "S3v2"}

// isValidSecretKey - validate secret key.
func isValidSecretKey(secretKey string) bool {
	if secretKey == "" {
		return true
	}
	regex := regexp.MustCompile(`.{8,40}$`)
	return regex.MatchString(secretKey) && !strings.ContainsAny(secretKey, "$%^~`!|&*#@")
}

// isValidAccessKey - validate access key.
func isValidAccessKey(accessKey string) bool {
	if accessKey == "" {
		return true
	}
	regex := regexp.MustCompile(`.{5,40}$`)
	return regex.MatchString(accessKey) && !strings.ContainsAny(accessKey, "$%^~`!|&*#@")
}

// isValidHostURL - validate input host url.
func isValidHostURL(hostURL string) bool {
	if strings.TrimSpace(hostURL) == "" {
		return false
	}
	url := newClientURL(hostURL)
	if url.Scheme != "https" && url.Scheme != "http" {
		return false
	}
	if url.Path != "" && url.Path != "/" {
		return false
	}
	return true
}

// isValidAPI - Validates if API signature string of supported type.
func isValidAPI(api string) bool {
	switch strings.ToLower(api) {
	case "s3v2", "s3v4":
		return true
	default:
		return false
	}
}
