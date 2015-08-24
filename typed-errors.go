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

import (
	"errors"

	"github.com/minio/minio/pkg/probe"
)

var (
	errDummy = func() *probe.Error {
		return probe.NewError(errors.New("")).Untrace()
	}
	errInvalidArgument = func() *probe.Error {
		return probe.NewError(errors.New("Invalid arguments provided, cannot proceed.")).Untrace()
	}
	errSourceListEmpty = func() *probe.Error {
		return probe.NewError(errors.New("Source argument list is empty.")).Untrace()
	}
)

type eInvalidGlobURL struct {
	glob    string
	request string
}

func (e eInvalidGlobURL) Error() string {
	return "Error reading glob URL ‘" + e.glob + "’ while comparing with ‘" + e.request + "’."
}

type eInvalidURL struct {
	URL string
}

func (e eInvalidURL) Error() string {
	return "Invalid URL " + e.URL
}

type eNoMatchingHost eInvalidURL

func (e eNoMatchingHost) Error() string {
	return "No matching host found for the given URL ‘" + e.URL + "’."
}

type eInitClient eInvalidURL

func (e eInitClient) Error() string {
	return "Unable to initialize client for URL ‘" + e.URL + "’."
}

type eInvalidSource eInvalidURL

func (e eInvalidSource) Error() string {
	return "Invalid source " + e.URL
}

type eInvalidTarget eInvalidURL

func (e eInvalidTarget) Error() string {
	return "Invalid target " + e.URL
}

type eSourceNotRecursive eInvalidURL

func (e eSourceNotRecursive) Error() string {
	return "Source ‘" + e.URL + "’ is not recursive."
}

type eSourceIsNotDir eInvalidURL

func (e eSourceIsNotDir) Error() string {
	return "Source ‘" + e.URL + "’ is not a folder."
}

type eNotAnObject eInvalidURL

func (e eNotAnObject) Error() string {
	return "‘" + e.URL + "’ is not an object."
}
