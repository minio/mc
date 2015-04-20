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

import "regexp"

// isValidSecretKey - validate secret key
func isValidSecretKey(secretAccessKey string) bool {
	regex := regexp.MustCompile("^.{40}$")
	return regex.MatchString(secretAccessKey)
}

// isValidAccessKey - validate access key
func isValidAccessKey(accessKeyID string) bool {
	regex := regexp.MustCompile("^[A-Z0-9\\-\\.\\_\\~]{20}$")
	regex.MatchString(accessKeyID)
	return regex.MatchString(accessKeyID)
}
