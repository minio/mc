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

// isValidBucketACL - is provided acl string supported
func (b bucketACL) isValidBucketACL() bool {
	switch true {
	case b.isPrivate():
		fallthrough
	case b.isReadOnly():
		fallthrough
	case b.isPublic():
		fallthrough
	case b.isAuthenticated():
		return true
	case b.String() == "private":
		// by default its "private"
		return true
	default:
		return false
	}
}

// bucketACL - bucket level access control
type bucketACL string

// different types of ACL's currently supported for buckets
const (
	bucketPrivate       = bucketACL("private")
	bucketReadOnly      = bucketACL("readonly")
	bucketPublic        = bucketACL("public")
	bucketAuthenticated = bucketACL("authenticated")
)

func (b bucketACL) String() string {
	if string(b) == "" {
		return "private"
	}
	if string(b) == "readonly" {
		return "public-read"
	}
	if string(b) == "public" {
		return "public-read-write"
	}
	if string(b) == "authenticated" {
		return "authenticated-read"
	}
	return string(b)
}

// IsPrivate - is acl Private
func (b bucketACL) isPrivate() bool {
	return b == bucketPrivate
}

// IsPublicRead - is acl PublicRead
func (b bucketACL) isReadOnly() bool {
	return b == bucketReadOnly
}

// IsPublicReadWrite - is acl PublicReadWrite
func (b bucketACL) isPublic() bool {
	return b == bucketPublic
}

// IsAuthenticated - is acl AuthenticatedRead
func (b bucketACL) isAuthenticated() bool {
	return b == bucketAuthenticated
}
