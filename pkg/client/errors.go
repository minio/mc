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

// InvalidRange - invalid range requested
type InvalidRange struct {
	Offset int64
}

func (e InvalidRange) Error() string {
	return "Invalid range offset: " + strconv.FormatInt(e.Offset, 10)
}

// GenericBucketError - generic bucket operations error
type GenericBucketError struct {
	Bucket string
}

// BucketExists - bucket exists
type BucketExists GenericBucketError

func (e BucketExists) Error() string {
	return "Bucket #" + e.Bucket + " exists"
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

// InvalidObjectName - object requested is invalid
type InvalidObjectName GenericObjectError

func (e InvalidObjectName) Error() string {
	return "Object #" + e.Object + " at " + e.Bucket + " is invalid"
}

// ObjectExists - object exists
type ObjectExists GenericObjectError

func (e ObjectExists) Error() string {
	return "Object #" + e.Object + " exists"
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

// GenericFileError - generic file error.
type GenericFileError struct {
	Path string
}

// PathNotFound (ENOENT) - file not found.
type PathNotFound GenericFileError

func (e PathNotFound) Error() string {
	return "Requested file ‘" + e.Path + "’ not found"
}

// PathInsufficientPermission (EPERM) - permission denied.
type PathInsufficientPermission GenericFileError

func (e PathInsufficientPermission) Error() string {
	return "Insufficient permissions to access this file ‘" + e.Path + "’"
}

// BrokenSymlink (ENOTENT) - file has broken symlink.
type BrokenSymlink GenericFileError

func (e BrokenSymlink) Error() string {
	return "Requested file ‘" + e.Path + "’ has broken symlink"
}

// TooManyLevelsSymlink (ELOOP) - file has too many levels of symlinks.
type TooManyLevelsSymlink GenericFileError

func (e TooManyLevelsSymlink) Error() string {
	return "Requested file ‘" + e.Path + "’ has too many levels of symlinks"
}

// EmptyPath (EINVAL) - invalid argument.
type EmptyPath struct{}

func (e EmptyPath) Error() string {
	return "Invalid path, path cannot be empty"
}

// ObjectMissing (EINVAL) - object key missing.
type ObjectMissing struct{}

func (e ObjectMissing) Error() string {
	return "Object key is missing, object key cannot be empty"
}
