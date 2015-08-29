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

import "github.com/minio/minio/pkg/probe"

// isValidAccessPERM - is provided acl string supported
func (b accessPerms) isValidAccessPERM() bool {
	switch true {
	case b.isPrivate():
		fallthrough
	case b.isReadOnly():
		fallthrough
	case b.isPublic():
		fallthrough
	case b.isAuthenticated():
		return true
	default:
		return false
	}
}

// accessPerms - access level access control
type accessPerms string

// different types of ACL's currently supported for accesss
const (
	accessPrivate       = accessPerms("private")
	accessReadOnly      = accessPerms("readonly")
	accessPublic        = accessPerms("public")
	accessAuthenticated = accessPerms("authenticated")
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
	if b.isAuthenticated() {
		return "authenticated-read"
	}
	return "private"
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

// IsAuthenticated - is acl AuthenticatedRead
func (b accessPerms) isAuthenticated() bool {
	return b == accessAuthenticated
}

// getDefaultAccess - read ACL from config
func getDefaultAccess() (acl string, err *probe.Error) {
	config, err := getMcConfig()
	if err != nil {
		return "", err.Trace()
	}
	return config.Access, nil
}
