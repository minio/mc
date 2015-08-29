/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

// ACL -  level access control
type ACL string

// different types of ACL's currently supported
const (
	Private       = ACL("private")
	ReadOnly      = ACL("public-read")
	Public        = ACL("public-read-write")
	Authenticated = ACL("authenticated-read")
)

// String printer helper
func (b ACL) String() string {
	if string(b) == "" {
		return "private"
	}
	return string(b)
}

// isValidACL - is provided acl string supported
func (b ACL) isValidACL() bool {
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

// IsPrivate - is acl Private
func (b ACL) isPrivate() bool {
	return b == Private
}

// IsPublicRead - is acl PublicRead
func (b ACL) isReadOnly() bool {
	return b == ReadOnly
}

// IsPublicReadWrite - is acl PublicReadWrite
func (b ACL) isPublic() bool {
	return b == Public
}

// IsAuthenticated - is acl AuthenticatedRead
func (b ACL) isAuthenticated() bool {
	return b == Authenticated
}
