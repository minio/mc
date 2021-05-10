// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"time"
)

/// Collection of standard errors

// APINotImplemented - api not implemented
type APINotImplemented struct {
	API     string
	APIType string
}

func (e APINotImplemented) Error() string {
	return "`" + e.API + "` is not supported for `" + e.APIType + "`."
}

// GenericBucketError - generic bucket operations error
type GenericBucketError struct {
	Bucket string
}

// BucketDoesNotExist - bucket does not exist.
type BucketDoesNotExist GenericBucketError

func (e BucketDoesNotExist) Error() string {
	return "Bucket `" + e.Bucket + "` does not exist."
}

// BucketExists - bucket exists.
type BucketExists GenericBucketError

func (e BucketExists) Error() string {
	return "Bucket `" + e.Bucket + "` exists."
}

// BucketNameEmpty - bucket name empty (http://goo.gl/wJlzDz)
type BucketNameEmpty struct{}

func (e BucketNameEmpty) Error() string {
	return "Bucket name cannot be empty."
}

// ObjectNameEmpty - object name empty.
type ObjectNameEmpty struct{}

func (e ObjectNameEmpty) Error() string {
	return "Object name cannot be empty."
}

// BucketInvalid - bucket name invalid.
type BucketInvalid struct {
	Bucket string
}

func (e BucketInvalid) Error() string {
	return "Bucket name " + e.Bucket + " not valid."
}

// ObjectAlreadyExists - typed return for MethodNotAllowed
type ObjectAlreadyExists struct {
	Object string
}

func (e ObjectAlreadyExists) Error() string {
	return "Object `" + e.Object + "` already exists."
}

// ObjectAlreadyExistsAsDirectory - typed return for XMinioObjectExistsAsDirectory
type ObjectAlreadyExistsAsDirectory struct {
	Object string
}

func (e ObjectAlreadyExistsAsDirectory) Error() string {
	return "Object `" + e.Object + "` already exists as directory."
}

// ObjectOnGlacier - object is of storage class glacier.
type ObjectOnGlacier struct {
	Object string
}

func (e ObjectOnGlacier) Error() string {
	return "Object `" + e.Object + "` is on Glacier storage."
}

// BucketNameTopLevel - generic error
type BucketNameTopLevel struct{}

func (e BucketNameTopLevel) Error() string {
	return "Buckets or prefixes can only be created with `/` suffix."
}

// GenericFileError - generic file error.
type GenericFileError struct {
	Path string
}

// PathNotFound (ENOENT) - file not found.
type PathNotFound GenericFileError

func (e PathNotFound) Error() string {
	return "Requested file `" + e.Path + "` not found"
}

// PathIsNotRegular (ENOTREG) - file is not a regular file.
type PathIsNotRegular GenericFileError

func (e PathIsNotRegular) Error() string {
	return "Requested file `" + e.Path + "` is not a regular file."
}

// PathInsufficientPermission (EPERM) - permission denied.
type PathInsufficientPermission GenericFileError

func (e PathInsufficientPermission) Error() string {
	return "Insufficient permissions to access this file `" + e.Path + "`"
}

// BrokenSymlink (ENOTENT) - file has broken symlink.
type BrokenSymlink GenericFileError

func (e BrokenSymlink) Error() string {
	return "Requested file `" + e.Path + "` has broken symlink"
}

// TooManyLevelsSymlink (ELOOP) - file has too many levels of symlinks.
type TooManyLevelsSymlink GenericFileError

func (e TooManyLevelsSymlink) Error() string {
	return "Requested file `" + e.Path + "` has too many levels of symlinks"
}

// EmptyPath (EINVAL) - invalid argument.
type EmptyPath struct{}

func (e EmptyPath) Error() string {
	return "Invalid path, path cannot be empty"
}

// ObjectMissing (EINVAL) - object key missing.
type ObjectMissing struct {
	timeRef time.Time
}

func (e ObjectMissing) Error() string {
	if !e.timeRef.IsZero() {
		return "Object did not exist at `" + e.timeRef.Format(time.RFC1123) + "`"
	}
	return "Object does not exist"
}

// ObjectIsDeleteMarker - object is a delete marker as latest
type ObjectIsDeleteMarker struct {
}

func (e ObjectIsDeleteMarker) Error() string {
	return "Object is marked as deleted"
}

// UnexpectedShortWrite - write wrote less bytes than expected.
type UnexpectedShortWrite struct {
	InputSize int
	WriteSize int
}

func (e UnexpectedShortWrite) Error() string {
	msg := fmt.Sprintf("Wrote less data than requested. Expected `%d` bytes, but only wrote `%d` bytes.", e.InputSize, e.WriteSize)
	return msg
}

// UnexpectedEOF (EPIPE) - reader closed prematurely.
type UnexpectedEOF struct {
	TotalSize    int64
	TotalWritten int64
}

func (e UnexpectedEOF) Error() string {
	msg := fmt.Sprintf("Input reader closed pre-maturely. Expected `%d` bytes, but only received `%d` bytes.", e.TotalSize, e.TotalWritten)
	return msg
}

// UnexpectedExcessRead - reader wrote more data than requested.
type UnexpectedExcessRead UnexpectedEOF

func (e UnexpectedExcessRead) Error() string {
	msg := fmt.Sprintf("Received excess data on input reader. Expected only `%d` bytes, but received `%d` bytes.", e.TotalSize, e.TotalWritten)
	return msg
}

// SameFile - source and destination are same files.
type SameFile struct {
	Source, Destination string
}

func (e SameFile) Error() string {
	return fmt.Sprintf("'%s' and '%s' are the same file", e.Source, e.Destination)
}
