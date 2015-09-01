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

// isValidBucketPERM - is provided acl string supported
func (b bucketPerms) isValidBucketPERM() bool {
	switch true {
	case b.isPrivate():
		fallthrough
	case b.isReadOnly():
		fallthrough
	case b.isPublic():
		fallthrough
	case b.isAuthorized():
		return true
	default:
		return false
	}
}

// bucketPerms - bucket level access control
type bucketPerms string

// different types of ACL's currently supported for buckets
const (
	bucketPrivate    = bucketPerms("private")
	bucketReadOnly   = bucketPerms("readonly")
	bucketPublic     = bucketPerms("public")
	bucketAuthorized = bucketPerms("authorized")
)

func (b bucketPerms) String() string {
	if !b.isValidBucketPERM() {
		return string(b)
	}
	if b.isReadOnly() {
		return "public-read"
	}
	if b.isPublic() {
		return "public-read-write"
	}
	if b.isAuthorized() {
		return "authenticated-read"
	}
	return "private"
}

func aclToPerms(acl string) bucketPerms {
	switch acl {
	case "private":
		return bucketPerms("private")
	case "public-read":
		return bucketPerms("readonly")
	case "public-read-write":
		return bucketPerms("public")
	case "authenticated-read":
		return bucketPerms("authorized")
	default:
		return bucketPerms(acl)
	}
}

// IsPrivate - is acl Private
func (b bucketPerms) isPrivate() bool {
	return b == bucketPrivate
}

// IsPublicRead - is acl PublicRead
func (b bucketPerms) isReadOnly() bool {
	return b == bucketReadOnly
}

// IsPublicReadWrite - is acl PublicReadWrite
func (b bucketPerms) isPublic() bool {
	return b == bucketPublic
}

// IsAuthorized - is acl AuthorizedRead
func (b bucketPerms) isAuthorized() bool {
	return b == bucketAuthorized
}
