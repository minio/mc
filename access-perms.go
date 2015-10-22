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

// isValidAccessPERM - is provided acl string supported
func (b accessPerms) isValidAccessPERM() bool {
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

// accessPerms - access level access control
type accessPerms string

// different types of ACL's currently supported for accesss
const (
	accessPrivate    = accessPerms("private")
	accessReadOnly   = accessPerms("readonly")
	accessPublic     = accessPerms("public")
	accessAuthorized = accessPerms("authorized")
)

func (b accessPerms) String() string {
	if !b.isValidAccessPERM() {
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

func aclToPerms(acl string) accessPerms {
	switch acl {
	case "private":
		return accessPerms("private")
	case "public-read":
		return accessPerms("readonly")
	case "public-read-write":
		return accessPerms("public")
	case "authenticated-read":
		return accessPerms("authorized")
	default:
		return accessPerms(acl)
	}
}

// IsPrivate - is acl Private
func (b accessPerms) isPrivate() bool {
	return b == accessPrivate
}

// IsPublicRead - is acl PublicRead
func (b accessPerms) isReadOnly() bool {
	return b == accessReadOnly
}

// IsPublicReadWrite - is acl PublicReadWrite
func (b accessPerms) isPublic() bool {
	return b == accessPublic
}

// IsAuthorized - is acl AuthorizedRead
func (b accessPerms) isAuthorized() bool {
	return b == accessAuthorized
}
