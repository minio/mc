/*
 * Minimalist Object Storage SDK (C) 2015 Minio, Inc.
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

package minio

import "strings"

func url2Bucket(url string) string {
	splits := strings.SplitN(url, "/", 2)
	switch len(splits) {
	case 2:
		return splits[1]
	default:
		return ""
	}
}

func isPrefixCaseInsensitive(s, pfx string) bool {
	if len(pfx) > len(s) {
		return false
	}
	shead := s[:len(pfx)]
	if shead == pfx {
		return true
	}
	shead = strings.ToLower(shead)
	return shead == pfx || shead == strings.ToLower(pfx)
}
