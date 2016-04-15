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

package main

import (
	"fmt"

	"github.com/minio/cli"
)

func checkCopySyntax(ctx *cli.Context) {
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

	/****** Generic Invalid Rules *******/
	// Verify if source(s) exists.
	for _, srcURL := range srcURLs {
		_, _, err := url2Stat(srcURL)
		if err != nil {
			fatalIf(err.Trace(srcURL), fmt.Sprintf("Unable to stat '%s'.", srcURL))
		}
	}

	// Check if bucket name is passed for URL type arguments.
	url := newClientURL(tgtURL)
	if url.Host != "" {
		// This check is for type URL.
		if !isURLVirtualHostStyle(url.Host) {
			if url.Path == string(url.Separator) {
				fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Target ‘%s’ does not contain bucket name.", tgtURL))
			}
		}
	}

	// Guess CopyURLsType based on source and target URLs.
	copyURLsType, err := guessCopyURLType(srcURLs, tgtURL, isRecursive)
	if err != nil {
		fatalIf(errInvalidArgument().Trace(), "Unable to guess the type of copy operation.")
	}
	switch copyURLsType {
	case copyURLsTypeA: // File -> File.
		checkCopySyntaxTypeA(srcURLs, tgtURL)
	case copyURLsTypeB: // File -> Folder.
		checkCopySyntaxTypeB(srcURLs, tgtURL)
	case copyURLsTypeC: // Folder... -> Folder.
		checkCopySyntaxTypeC(srcURLs, tgtURL, isRecursive)
	case copyURLsTypeD: // File1...FileN -> Folder.
		checkCopySyntaxTypeD(srcURLs, tgtURL)
	default:
		fatalIf(errInvalidArgument().Trace(), "Unable to guess the type of copy operation.")
	}
}

// checkCopySyntaxTypeA verifies if the source and target are valid file arguments.
func checkCopySyntaxTypeA(srcURLs []string, tgtURL string) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}
	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL)
	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")

	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(), "Source ‘"+srcURL+"’ is not a file.")
	}
}

// checkCopySyntaxTypeB verifies if the source is a valid file and target is a valid folder.
func checkCopySyntaxTypeB(srcURLs []string, tgtURL string) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}
	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL)
	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")

	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(srcURL), "Source ‘"+srcURL+"’ is not a file.")
	}

	// Check target.
	if _, tgtContent, err := url2Stat(tgtURL); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}

// checkCopySyntaxTypeC verifies if the source is a valid recursive dir and target is a valid folder.
func checkCopySyntaxTypeC(srcURLs []string, tgtURL string, isRecursive bool) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}

	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL)
	// incomplete uploads are not necessary for copy operation, no need to verify for them.
	isIncomplete := false
	if err != nil && !isURLPrefixExists(srcURL, isIncomplete) {
		fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")
	}

	if srcContent.Type.IsDir() && !isRecursive {
		fatalIf(errInvalidArgument().Trace(srcURL), "To copy a folder requires --recursive option.")
	}

	// Check target.
	if _, tgtContent, err := url2Stat(tgtURL); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}

// checkCopySyntaxTypeD verifies if the source is a valid list of files and target is a valid folder.
func checkCopySyntaxTypeD(srcURLs []string, tgtURL string) {
	// Source can be anything: file, dir, dir...
	// Check target if it is a dir
	if _, tgtContent, err := url2Stat(tgtURL); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}
