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

// InvalidMaxKeys - invalid maxkeys provided
type InvalidMaxKeys struct {
	MaxKeys int
}

func (e InvalidMaxKeys) Error() string {
	return "invalid maxkeys: " + strconv.Itoa(e.MaxKeys)
}

// InvalidRange - invalid range requested
type InvalidRange struct {
	Offset int64
}

func (e InvalidRange) Error() string {
	return "invalid range offset: " + strconv.FormatInt(e.Offset, 10)
}

// GenericBucketError - generic bucket operations error
type GenericBucketError struct {
	Bucket string
}

// BucketNotFound - bucket requested does not exist
type BucketNotFound GenericBucketError

func (e BucketNotFound) Error() string {
	return "bucket " + e.Bucket + " not found"
}

// BucketExists - bucket exists
type BucketExists GenericBucketError

func (e BucketExists) Error() string {
	return "bucket " + e.Bucket + " exists"
}

// InvalidBucketName - bucket name invalid
type InvalidBucketName GenericBucketError

func (e InvalidBucketName) Error() string {
	return "Invalid bucketname " + e.Bucket
}

// GenericObjectError - generic object operations error
type GenericObjectError struct {
	Bucket string
	Object string
}

// ObjectNotFound - object requested does not exist
type ObjectNotFound GenericObjectError

func (e ObjectNotFound) Error() string {
	return "object " + e.Object + " not found in bucket " + e.Bucket
}

// InvalidObjectName - object requested is invalid
type InvalidObjectName GenericObjectError

func (e InvalidObjectName) Error() string {
	return "object " + e.Object + "at" + e.Bucket + "is invalid"
}

// ObjectExists - object exists
type ObjectExists GenericObjectError

func (e ObjectExists) Error() string {
	return "object " + e.Object + " exists"
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

// InvalidAuthorizationKey - invalid authorization key
type InvalidAuthorizationKey GenericError

func (e InvalidAuthorizationKey) Error() string {
	return e.Err.Error()
}

// AuthorizationKeyEmpty - empty auth key provided
type AuthorizationKeyEmpty GenericError

func (e AuthorizationKeyEmpty) Error() string {
	return e.Err.Error()
}
