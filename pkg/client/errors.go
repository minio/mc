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
	API     string
	APIType string
}

func (e APINotImplemented) Error() string {
	return "‘" + e.API + "’ feature " + "is not implemented for ‘" + e.APIType + "’"
}

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

// InvalidMaxKeys - invalid maxkeys provided
type InvalidMaxKeys struct {
	MaxKeys int
}

func (e InvalidMaxKeys) Error() string {
	return "invalid maxkeys: " + strconv.Itoa(e.MaxKeys)
}

// GenericBucketError - generic bucket operations error
type GenericBucketError struct {
	Bucket string
}

// BucketExists - bucket exists
type BucketExists GenericBucketError

func (e BucketExists) Error() string {
	return "bucket " + e.Bucket + " exists"
}

// InvalidBucketName - bucket name invalid (http://goo.gl/wJlzDz)
type InvalidBucketName GenericBucketError

func (e InvalidBucketName) Error() string {
	return "Invalid bucketname [" + e.Bucket + "], please read http://goo.gl/wJlzDz"
}

// GenericObjectError - generic object operations error
type GenericObjectError struct {
	Bucket string
	Object string
}

// ObjectAlreadyExists - typed return for MethodNotAllowed
type ObjectAlreadyExists struct {
	Object string
}

func (e ObjectAlreadyExists) Error() string {
	return "Object #" + e.Object + " already exists."
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
type GenericError struct{}

// InvalidQueryURL - generic error
type InvalidQueryURL struct {
	URL string
}

func (e InvalidQueryURL) Error() string {
	return "Invalid query URL: " + e.URL
}

// GenericFileError - generic file error
type GenericFileError struct {
	Path string
}

// NotFound (ENOENT) - file not found
type NotFound GenericFileError

func (e NotFound) Error() string {
	return "Requested file ‘" + e.Path + "’ not found"
}

// BrokenSymlink (ENOTENT) - file has broken symlink
type BrokenSymlink GenericFileError

func (e BrokenSymlink) Error() string {
	return "Requested file ‘" + e.Path + "’ has broken symlink"
}

// TooManyLevelsSymlink (ELOOP) - file has too many levels of symlinks
type TooManyLevelsSymlink GenericFileError

func (e TooManyLevelsSymlink) Error() string {
	return "Requested file ‘" + e.Path + "’ has too many levels of symlinks"
}

// EmptyPath (EINVAL) - invalid argument
type EmptyPath struct{}

func (e EmptyPath) Error() string {
	return "Invalid path, path cannot be empty"
}
