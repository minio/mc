/*
 * Modern Copy, (C) 2014,2015 Minio, Inc.
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

import "strconv"

type errEmptyURL struct{}

func (e errEmptyURL) Error() string {
	return "URL is empty"
}

type errUnsupportedScheme struct {
	scheme urlType
}

func (e errUnsupportedScheme) Error() string {
	return "Unsuppported URL scheme: " + strconv.Itoa(int(e.scheme))
}

type errInvalidURL struct {
	url string
}

func (e errInvalidURL) Error() string {
	return "Invalid URL: " + e.url
}

type errInvalidBucket struct {
	bucket string
}

func (e errInvalidBucket) Error() string {
	return "Invalid bucket name: " + e.bucket
}
