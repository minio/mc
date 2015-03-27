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

func isStringInSlice(items []string, item string) bool {
	for _, s := range items {
		if s == item {
			return true
		}
	}
	return false
}

func deleteFromSlice(items []string, item string) []string {
	var newitems []string
	for _, s := range items {
		if s == item {
			continue
		}
		newitems = append(newitems, s)
	}
	return newitems
}

func appendUniq(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

// Is alphanumeric?
func isalnum(c rune) bool {
	return '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

// isValidDonutName - verify donutName to be valid
func isValidDonutName(donutName string) bool {
	if len(donutName) > 1024 || len(donutName) == 0 {
		return false
	}
	for _, char := range donutName {
		if isalnum(char) {
			continue
		}
		switch char {
		case '-':
		case '.':
		case '_':
		case '~':
			continue
		default:
			return false
		}
	}
	return true
}
