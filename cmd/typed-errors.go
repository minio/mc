/*
 * MinIO Client (C) 2014, 2015, 2018 MinIO, Inc.
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

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/minio/mc/pkg/probe"
)

type dummyErr error

var errDummy = func() *probe.Error {
	msg := ""
	return probe.NewError(dummyErr(errors.New(msg))).Untrace()
}

type invalidArgumentErr error

var errInvalidArgument = func() *probe.Error {
	msg := "Invalid arguments provided, please refer " + "`mc <command> -h` for relevant documentation."
	return probe.NewError(invalidArgumentErr(errors.New(msg))).Untrace()
}

type unrecognizedDiffTypeErr error

var errUnrecognizedDiffType = func(diff differType) *probe.Error {
	msg := "Unrecognized diffType: " + diff.String() + " provided."
	return probe.NewError(unrecognizedDiffTypeErr(errors.New(msg))).Untrace()
}

type invalidAliasedURLErr error

var errInvalidAliasedURL = func(URL string) *probe.Error {
	msg := "Use `mc alias set mycloud " + URL + " ...` to add an alias. Use the alias for S3 operations."
	return probe.NewError(invalidAliasedURLErr(errors.New(msg))).Untrace()
}

type invalidAliasErr error

var errInvalidAlias = func(alias string) *probe.Error {
	msg := "Alias `" + alias + "` should have alphanumeric characters such as [helloWorld0, hello_World0, ...]"
	return probe.NewError(invalidAliasErr(errors.New(msg)))
}

type invalidURLErr error

var errInvalidURL = func(URL string) *probe.Error {
	msg := "URL `" + URL + "` for MinIO Client should be of the form scheme://host[:port]/ without resource component."
	return probe.NewError(invalidURLErr(errors.New(msg)))
}

type invalidAPISignatureErr error

var errInvalidAPISignature = func(api, url string) *probe.Error {
	msg := fmt.Sprintf(
		"Unrecognized API signature %s for host %s. Valid options are `[%s]`",
		api, url, strings.Join(validAPIs, ", "))
	return probe.NewError(invalidAPISignatureErr(errors.New(msg)))
}

type noMatchingHostErr error

var errNoMatchingHost = func(URL string) *probe.Error {
	msg := "No matching host found for the given URL `" + URL + "`."
	return probe.NewError(noMatchingHostErr(errors.New(msg))).Untrace()
}

type invalidSourceErr error

var errInvalidSource = func(URL string) *probe.Error {
	msg := "Invalid source `" + URL + "`."
	return probe.NewError(invalidSourceErr(errors.New(msg))).Untrace()
}

type invalidTargetErr error

var errInvalidTarget = func(URL string) *probe.Error {
	msg := "Invalid target `" + URL + "`."
	return probe.NewError(invalidTargetErr(errors.New(msg))).Untrace()
}

type targetNotFoundErr error

var errTargetNotFound = func(URL string) *probe.Error {
	msg := "Target `" + URL + "` not found."
	return probe.NewError(targetNotFoundErr(errors.New(msg))).Untrace()
}

type overwriteNotAllowedErr struct {
	error
}

var errOverWriteNotAllowed = func(URL string) *probe.Error {
	msg := "Overwrite not allowed for `" + URL + "`. Use `--overwrite` to override this behavior."
	return probe.NewError(overwriteNotAllowedErr{errors.New(msg)})
}

type sourceIsDirErr error

var errSourceIsDir = func(URL string) *probe.Error {
	msg := "Source `" + URL + "` is a folder."
	return probe.NewError(sourceIsDirErr(errors.New(msg))).Untrace()
}

type conflictSSEErr error

var errConflictSSE = func(sseServer, sseKeys string) *probe.Error {
	err := fmt.Errorf("SSE alias '%s' overlaps with SSE-C aliases '%s'", sseServer, sseKeys)
	return probe.NewError(conflictSSEErr(err)).Untrace()
}
