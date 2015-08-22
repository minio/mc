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

import "errors"

type errNotAnObject struct {
	url string
}

func (e errNotAnObject) Error() string {
	return "URL: ‘" + e.url + "’ not an object"
}

var errInvalidArgument = errors.New("Invalid arguments provided, cannot proceed.")

type errInvalidGlobURL struct {
	glob    string
	request string
}

func (e errInvalidGlobURL) Error() string {
	return "Error reading glob URL ‘" + e.glob + "’ while comparing with ‘" + e.request + "’."
}

type errNoMatchingHost struct {
	url string
}

func (e errNoMatchingHost) Error() string {
	return "No matching host found for the given url ‘" + e.url + "’."
}

type errInitClient struct {
	url string
}

func (e errInitClient) Error() string {
	return "Unable to initialize client for url ‘" + e.url + "’."
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

var errSourceListEmpty = errors.New("Source list is empty.")
