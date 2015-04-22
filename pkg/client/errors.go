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
type GenericError struct {
	Err error
}

// UnexpectedError - unexpected error
type UnexpectedError GenericError

func (e UnexpectedError) Error() string {
	return e.Err.Error() + ", please report this error"
}

// InvalidArgument - bad arguments provided
type InvalidArgument GenericError

func (e InvalidArgument) Error() string {
	return e.Err.Error()
}

// InvalidRange - invalid range requested
type InvalidRange struct {
	Offset int64
}

func (e InvalidRange) Error() string {
	return "invalid range offset: " + strconv.FormatInt(e.Offset, 10)
}
