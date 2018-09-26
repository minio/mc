/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"fmt"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

func checkCopySyntax(ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code.
	}

	// extract URLs.
	URLs := ctx.Args()
	if len(URLs) < 2 {
		fatalIf(errDummy().Trace(ctx.Args()...), fmt.Sprintf("Unable to parse source and target arguments."))
	}

	srcURLs := URLs[:len(URLs)-1]
	tgtURL := URLs[len(URLs)-1]
	isRecursive := ctx.Bool("recursive")

	// Verify if source(s) exists.
	for _, srcURL := range srcURLs {
		_, _, err := url2Stat(srcURL, false, encKeyDB)
		if err != nil {
			console.Fatalf("Unable to validate source %s\n", srcURL)
		}
	}

	// Check if bucket name is passed for URL type arguments.
	url := newClientURL(tgtURL)
	if url.Host != "" {
		if url.Path == string(url.Separator) {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Target `%s` does not contain bucket name.", tgtURL))
		}
	}

	// Guess CopyURLsType based on source and target URLs.
	copyURLsType, err := guessCopyURLType(srcURLs, tgtURL, isRecursive, encKeyDB)
	if err != nil {
		fatalIf(errInvalidArgument().Trace(), "Unable to guess the type of copy operation.")
	}

	switch copyURLsType {
	case copyURLsTypeA: // File -> File.
		checkCopySyntaxTypeA(srcURLs, tgtURL, encKeyDB)
	case copyURLsTypeB: // File -> Folder.
		checkCopySyntaxTypeB(srcURLs, tgtURL, encKeyDB)
	case copyURLsTypeC: // Folder... -> Folder.
		checkCopySyntaxTypeC(srcURLs, tgtURL, isRecursive, encKeyDB)
	case copyURLsTypeD: // File1...FileN -> Folder.
		checkCopySyntaxTypeD(srcURLs, tgtURL, encKeyDB)
	default:
		fatalIf(errInvalidArgument().Trace(), "Unable to guess the type of copy operation.")
	}
}

// checkCopySyntaxTypeA verifies if the source and target are valid file arguments.
func checkCopySyntaxTypeA(srcURLs []string, tgtURL string, keys map[string][]prefixSSEPair) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}
	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL, false, keys)
	fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")

	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(), "Source `"+srcURL+"` is not a file.")
	}
}

// checkCopySyntaxTypeB verifies if the source is a valid file and target is a valid folder.
func checkCopySyntaxTypeB(srcURLs []string, tgtURL string, keys map[string][]prefixSSEPair) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}
	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL, false, keys)
	fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")

	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(srcURL), "Source `"+srcURL+"` is not a file.")
	}

	// Check target.
	if _, tgtContent, err := url2Stat(tgtURL, false, keys); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target `"+tgtURL+"` is not a folder.")
		}
	}
}

// checkCopySyntaxTypeC verifies if the source is a valid recursive dir and target is a valid folder.
func checkCopySyntaxTypeC(srcURLs []string, tgtURL string, isRecursive bool, keys map[string][]prefixSSEPair) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}

	// Check target.
	if _, tgtContent, err := url2Stat(tgtURL, false, keys); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target `"+tgtURL+"` is not a folder.")
		}
	}

	for _, srcURL := range srcURLs {
		c, srcContent, err := url2Stat(srcURL, false, keys)
		// incomplete uploads are not necessary for copy operation, no need to verify for them.
		isIncomplete := false
		if err != nil {
			if !isURLPrefixExists(srcURL, isIncomplete) {
				fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")
			}
			// No more check here, continue to the next source url
			continue
		}

		if srcContent.Type.IsDir() {
			// Require --recursive flag if we are copying a directory
			if !isRecursive {
				fatalIf(errInvalidArgument().Trace(srcURL), "To copy a folder requires --recursive flag.")
			}

			// Check if we are going to copy a directory into itself
			if isURLContains(srcURL, tgtURL, string(c.GetURL().Separator)) {
				fatalIf(errInvalidArgument().Trace(), "Copying a folder into itself is not allowed.")
			}
		}
	}

}

// checkCopySyntaxTypeD verifies if the source is a valid list of files and target is a valid folder.
func checkCopySyntaxTypeD(srcURLs []string, tgtURL string, keys map[string][]prefixSSEPair) {
	// Source can be anything: file, dir, dir...
	// Check target if it is a dir
	if _, tgtContent, err := url2Stat(tgtURL, false, keys); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target `"+tgtURL+"` is not a folder.")
		}
	}
}
