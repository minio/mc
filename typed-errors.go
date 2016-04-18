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

	errUnrecognizedDiffType = func(diff differType) *probe.Error {
		return probe.NewError(errors.New("Unrecognized diffType: " + diff.String() + " provided.")).Untrace()
	}

	errInvalidAliasedURL = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Use ‘mc config host add mycloud " + URL + " ...’ to add an alias. Use the alias for S3 operations.")).Untrace()
	}

	errNoMatchingHost = func(URL string) *probe.Error {
		return probe.NewError(errors.New("No matching host found for the given URL ‘" + URL + "’.")).Untrace()
	}

	errInvalidSource = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Invalid source ‘" + URL + "’.")).Untrace()
	}

	errInvalidTarget = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Invalid target ‘" + URL + "’.")).Untrace()
	}

	errOverWriteNotAllowed = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Overwrite not allowed for ‘" + URL + "’. Use ‘--force’ to override this behavior."))
	}

	errDeleteNotAllowed = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Delete not allowed for ‘" + URL + "’. Use ‘--force’ to override this behavior."))
	}
	errSourceIsDir = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Source ‘" + URL + "’ is a folder.")).Untrace()
	}

	errSourceTargetSame = func(URL string) *probe.Error {
		return probe.NewError(errors.New("Source and target URL can not be same : " + URL)).Untrace()
	}
)
