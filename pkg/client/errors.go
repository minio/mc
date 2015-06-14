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

package client

import "strconv"

/// Collection of standard errors

// APINotImplemented - api not implemented
type APINotImplemented struct {
	API string
}

func (e APINotImplemented) Error() string {
	return "API not implemented: " + e.API
}

// GenericError - generic error
type GenericError struct{}

// UnexpectedError - unexpected error
type UnexpectedError GenericError

func (e UnexpectedError) Error() string {
	return "Unexpected error, please report this error at https://github.com/minio/mc/issues"
}

// InvalidArgument - bad arguments provided
type InvalidArgument GenericError

func (e InvalidArgument) Error() string {
	return "Invalid argument"
}

// InvalidRange - invalid range requested
type InvalidRange struct {
	Offset int64
}

func (e InvalidRange) Error() string {
	return "invalid range offset: " + strconv.FormatInt(e.Offset, 10)
}

// InvalidACLType - invalid acl type
type InvalidACLType struct {
	ACL string
}

func (e InvalidACLType) Error() string {
	return "invalid acl type: " + e.ACL
}
