/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

type errNotAnObject struct {
	url string
}

func (e errNotAnObject) Error() string {
	return "Not an object " + e.url
}

type errInvalidArgument struct{}

func (e errInvalidArgument) Error() string {
	return "Invalid argument."
}

type errUnsupportedScheme struct {
	scheme string
	url    string
}

func (e errUnsupportedScheme) Error() string {
	return "Unsuppported URL scheme: " + e.scheme
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
	return "Error reading glob URL " + e.glob + " while comparing with " + e.request
}

type errInvalidAliasName struct {
	name string
}

func (e errInvalidAliasName) Error() string {
	return "Not a valid alias name: " + e.name + " valid examples are: Area51, Grand-Nagus.."
}

type errInvalidAuth struct{}

func (e errInvalidAuth) Error() string {
	return "Invalid auth keys"
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
	return "Alias name: " + e.name + " exists"
}

/*
type errIsNotBucket struct {
	URL string
}

func (e errIsNotBucket) Error() string {
	return "Not a bucket " + e.URL
}

// errInvalidAuthKeys - invalid authorization keys
type errInvalidAuthKeys struct {
}

func (e errInvalidAuthKeys) Error() string {
	return "Invalid authorization keys"
}
*/

type errInvalidSource struct {
	URL string
}

func (e errInvalidSource) Error() string {
	return "Invalid source " + e.URL
}

type errInvalidTarget struct {
	URL string
}

func (e errInvalidTarget) Error() string {
	return "Invalid target " + e.URL
}
