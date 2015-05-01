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

import (
	"regexp"
	"unicode/utf8"
)

// IsValidBucketName reports whether bucket is a valid bucket name, per Amazon's naming restrictions.
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
func IsValidBucketName(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if bucket[0] == '.' || bucket[len(bucket)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucket); match == true {
		return false
	}
	// We don't support buckets with '.' in them, since the TLS requests will fail for not being
	// able to validate certificates
	match, _ := regexp.MatchString("^[a-z][a-z0-9\\-]+[a-z0-9]$", bucket)
	return match
}

// IsValidObjectName - verify object name in accordance with
//   - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
func IsValidObjectName(object string) bool {
	if len(object) > 1024 || len(object) == 0 {
		return false
	}
	if !utf8.ValidString(object) {
		return false
	}
	return true
}
