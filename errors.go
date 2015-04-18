/*
 * Mini Copy, (C) 2014,2015 Minio, Inc.
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

import (
	"fmt"

	"github.com/minio-io/mc/pkg/client"
)

type errInvalidArgument struct{}

func (e errInvalidArgument) Error() string {
	return "invalid argument"
}

type errUnsupportedScheme struct {
	scheme client.URLType
}

func (e errUnsupportedScheme) Error() string {
	return fmt.Sprintf("Unsuppported URL scheme: %d", e.scheme)
}

type errInvalidURL struct {
	url string
}

func (e errInvalidURL) Error() string {
	return "Invalid URL: " + e.url
}

type errInvalidGlobURL struct {
	glob    string
	request string
}

func (e errInvalidGlobURL) Error() string {
	return "Error parsing glob'ed URL while comparing " + e.glob + " " + e.request
}

type errInvalidAliasName struct {
	name string
}

func (e errInvalidAliasName) Error() string {
	return "Not a valid alias name: " + e.name + " Valid examples are: Area51, Grand-Nagus.."
}

type errInvalidAuth struct{}

func (e errInvalidAuth) Error() string {
	return "invalid auth keys"
}

type errNoMatchingHost struct{}

func (e errNoMatchingHost) Error() string {
	return "No matching host found."
}

type errConfigExists struct{}

func (e errConfigExists) Error() string {
	return "Config exists"
}

// errAliasExists - alias exists
type errAliasExists struct {
	name string
}

func (e errAliasExists) Error() string {
	return fmt.Sprintf("alias: %s exists", e.name)
}

// errInvalidAuthKeys - invalid authorization keys
type errInvalidAuthKeys struct{}

func (e errInvalidAuthKeys) Error() string {
	return "invalid authorization keys"
}

// errInvalidBucketName - invalid bucket name
type errInvalidBucketName struct {
	bucket string
}

func (e errInvalidBucketName) Error() string {
	return "invalid bucket name: " + e.bucket
}

type errBucketNameEmpty struct{}

func (e errBucketNameEmpty) Error() string {
	return "bucket name empty"
}
