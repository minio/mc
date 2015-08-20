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

type errUnexpected struct{}

func (e errUnexpected) Error() string {
	return "Unexpected control flow, please report this bug at https://github.com/minio/mc/issues."
}

type errInvalidSessionID struct {
	id string
}

func (e errInvalidSessionID) Error() string {
	return "Invalid session id ‘" + e.id + "’."
}

type errInvalidACL struct {
	acl string
}

func (e errInvalidACL) Error() string {
	return "Invalid ACL Type ‘" + e.acl + "’. Valid types are [private, public, readonly]."
}

type errNotAnObject struct {
	url string
}

func (e errNotAnObject) Error() string {
	return "URL: ‘" + e.url + "’ not an object"
}

type errInvalidArgument struct{}

func (e errInvalidArgument) Error() string {
	return "Incorrect usage, please use \"mc config help\""
}

type errInvalidGlobURL struct {
	glob    string
	request string
}

func (e errInvalidGlobURL) Error() string {
	return "Error reading glob URL " + e.glob + " while comparing with " + e.request
}

type errReservedAliasName struct {
	alias string
}

func (e errReservedAliasName) Error() string {
	return "Alias name ‘" + e.alias + "’ is a reserved word, reserved words are [help, private, readonly, public, authenticated]"
}

type errInvalidAliasName struct {
	alias string
}

func (e errInvalidAliasName) Error() string {
	return "Alias name ‘" + e.alias + "’ is invalid, valid examples are: Area51, Grand-Nagus.."
}

type errNoMatchingHost struct {
	url string
}

func (e errNoMatchingHost) Error() string {
	return "No matching host found for the given url ‘" + e.url + "’"
}

// errAliasExists - alias exists
type errAliasExists struct {
	alias string
}

func (e errAliasExists) Error() string {
	return "Alias name: ‘" + e.alias + "’ already exists."
}

type errInitClient struct {
	url string
}

func (e errInitClient) Error() string {
	return "Unable to initialize client for url ‘" + e.url + "’"
}

type errInvalidURL struct {
	URL string
}

func (e errInvalidURL) Error() string {
	return "Invalid url " + e.URL
}

type errInvalidSource errInvalidURL

func (e errInvalidSource) Error() string {
	return "Invalid source " + e.URL
}

type errInvalidTarget errInvalidURL

func (e errInvalidTarget) Error() string {
	return "Invalid target " + e.URL
}

type errTargetIsNotDir errInvalidURL

func (e errTargetIsNotDir) Error() string {
	return "Target ‘" + e.URL + "’ is not a folder."
}

type errSourceNotRecursive errInvalidURL

func (e errSourceNotRecursive) Error() string {
	return "Source ‘" + e.URL + "’ is not recursive."
}

type errSourceIsNotDir errTargetIsNotDir

func (e errSourceIsNotDir) Error() string {
	return "Source ‘" + e.URL + "’ is not a folder."
}

type errSourceIsNotFile errTargetIsNotDir

func (e errSourceIsNotFile) Error() string {
	return "Source ‘" + e.URL + "’ is not a file."
}

type errSourceListEmpty errInvalidArgument

func (e errSourceListEmpty) Error() string {
	return "Source list is empty."
}
