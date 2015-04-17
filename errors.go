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

import "fmt"

type errInvalidArgument struct{}

func (e errInvalidArgument) Error() string {
	return "invalid argument"
}

type errEmptyURL struct{}

func (e errEmptyURL) Error() string {
	return "URL is empty"
}

type errUnsupportedScheme struct {
	scheme urlType
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

type errInvalidAliasURL struct {
	url   string
	alias string
}

func (e errInvalidAliasURL) Error() string {
	return "Unable to parse URL: " + e.url + " for alias: " + e.alias
}

type errInvalidAliasName struct {
	alias string
}

func (e errInvalidAliasName) Error() string {
	return "Not a valid alias name: " + e.alias + " Valid examples are: Area51, Grand-Nagus.."
}

type errInvalidAuth struct{}

func (e errInvalidAuth) Error() string {
	return "invalid auth keys"
}

type errUnsupportedVersion struct {
	new uint
	old uint
}

func (e errUnsupportedVersion) Error() string {
	return fmt.Sprintf("Unsupported version [%d]. Current operating version is [%d]", e.new, e.old)
}

type errNoMatchingHost struct{}

func (e errNoMatchingHost) Error() string {
	return "No matching host found."
}
