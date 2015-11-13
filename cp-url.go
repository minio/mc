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
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
)

type copyURLs struct {
	SourceContent *client.Content
	TargetContent *client.Content
	Error         *probe.Error `json:"-"`
}

type copyURLsType uint8

const (
	copyURLsTypeInvalid copyURLsType = iota
	copyURLsTypeA                    // file to file
	copyURLsTypeB                    // file to dir
	copyURLsTypeC                    // recursive to dir
	copyURLsTypeD                    // complex to dir
)

//   NOTE: All the parse rules should reduced to A: Copy(Source, Target).
//
//   * VALID RULES
//   =======================
//   A: copy(f, f) -> copy(f, f)
//   B: copy(f, d) -> copy(f, d/f) -> A
//   C: copy(d1..., d2) -> []copy(d1/f, d2/d1/f) -> []A
//   D: copy([]{d1... | f}, d2) -> []{copy(d1/f, d2/d1/f) | copy(f, d2/f )} -> []A
//
//   * INVALID RULES
//   =========================
//   A: copy(d, *)
//   B: copy(d..., f)
//   C: copy(*, d...)
//
func checkCopySyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code.
	}
	// extract URLs.
	URLs, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(ctx.Args()...), fmt.Sprintf("One or more unknown URL types passed."))

	srcURLs := URLs[:len(URLs)-1]
	tgtURL := URLs[len(URLs)-1]

	/****** Generic rules *******/
	// Recursive URLs are not allowed in target.
	if isURLRecursive(tgtURL) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Recursive option is not supported for target ‘%s’ argument.", tgtURL))
	}
	url := client.NewURL(tgtURL)
	if url.Host != "" {
		// This check is for type URL.
		if url.Path == string(url.Separator) {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Target ‘%s’ does not contain bucket name.", tgtURL))
		}
	}
	switch guessCopyURLType(srcURLs, tgtURL) {
	case copyURLsTypeA: // File -> File.
		checkCopySyntaxTypeA(srcURLs, tgtURL)
	case copyURLsTypeB: // File -> Folder.
		checkCopySyntaxTypeB(srcURLs, tgtURL)
	case copyURLsTypeC: // Folder... -> Folder.
		checkCopySyntaxTypeC(srcURLs, tgtURL)
	case copyURLsTypeD: // File | Folder... -> Folder.
		checkCopySyntaxTypeD(srcURLs, tgtURL)
	default:
		fatalIf(errInvalidArgument().Trace(), "Invalid arguments to copy command.")
	}
}

// checkCopySyntaxTypeA verifies if the source and target are valid file arguments.
func checkCopySyntaxTypeA(srcURLs []string, tgtURL string) {
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments to copy command.")
	}
	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL)

	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")

	if srcContent.Type.IsDir() {
		fatalIf(errSourceIsDir(srcURL).Trace(srcURL), fmt.Sprintf("Folder cannot copied. Please use ‘%s...’ to copy this folder and its contents recursively.", srcURL))
	}
	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(), "Source ‘"+srcURL+"’ is not a file.")
	}
}

// checkCopySyntaxTypeB verifies if the source is a valid file and target is a valid dir.
func checkCopySyntaxTypeB(srcURLs []string, tgtURL string) {
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments to copy command.")
	}
	srcURL := srcURLs[0]
	_, srcContent, err := url2Stat(srcURL)
	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")

	if srcContent.Type.IsDir() {
		fatalIf(errSourceIsDir(srcURL).Trace(srcURL), fmt.Sprintf("Folder cannot be copied. Please use ‘%s...’ argument to copy this folder and its contents recursively.", srcURL))
	}
	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(srcURL), "Source ‘"+srcURL+"’ is not a file.")
	}

	_, tgtContent, err := url2Stat(tgtURL)
	// Target exist?.
	if err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}

// checkCopySyntaxTypeC verifies if the source is a valid recursive dir and target is a valid dir.
func checkCopySyntaxTypeC(srcURLs []string, tgtURL string) {
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments to copy command.")
	}

	srcURL := srcURLs[0]
	srcURL = stripRecursiveURL(srcURL)

	srcContent, err := url2Content(srcURL)
	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")

	if srcContent.Type.IsRegular() { // Ellipses is supported only for folders.
		fatalIf(errInvalidArgument().Trace(), "Source ‘"+srcURL+"’ is not a folder.")
	}
	_, tgtContent, err := url2Stat(tgtURL)
	// Target exist?.
	if err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}

// checkCopySyntaxTypeD verifies if the source is a valid list of file or valid recursive dir and target is a valid dir.
func checkCopySyntaxTypeD(srcURLs []string, tgtURL string) {
	for _, srcURL := range srcURLs {
		if isURLRecursive(srcURL) {
			srcURL = stripRecursiveURL(srcURL)
			_, srcContent, err := url2Stat(srcURL)
			fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")

			if !srcContent.Type.IsDir() { // Ellipses is supported only for folders.
				fatalIf(errInvalidArgument().Trace(srcURL), "Source ‘"+srcURL+"’ is not a folder.")
			}
		} else { // Regular URL.
			_, _, err := url2Stat(srcURL)
			fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")
		}
	}
	_, tgtContent, err := url2Stat(tgtURL)
	// Target exist?.
	if err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}

// guessCopyURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessCopyURLType(sourceURLs []string, targetURL string) copyURLsType {
	if strings.TrimSpace(targetURL) == "" || targetURL == "" { // Target is empty
		return copyURLsTypeInvalid
	}
	if len(sourceURLs) == 0 || sourceURLs == nil { // Source list is empty
		return copyURLsTypeInvalid
	}
	for _, sourceURL := range sourceURLs {
		if sourceURL == "" { // One of the source is empty
			return copyURLsTypeInvalid
		}
	}
	if len(sourceURLs) == 1 { // 1 Source, 1 Target
		switch {
		// Type C
		case isURLRecursive(sourceURLs[0]):
			return copyURLsTypeC
		// Type B
		case isTargetURLDir(targetURL):
			return copyURLsTypeB
		// Type A
		default:
			return copyURLsTypeA
		}
	} // else Type D
	return copyURLsTypeD
}

// SINGLE SOURCE - Type A: copy(f, f) -> copy(f, f)
// prepareCopyURLsTypeA - prepares target and source URLs for copying.
func prepareCopyURLsTypeA(sourceURL string, targetURL string) copyURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return copyURLs{Error: err.Trace(sourceURL)}
	}
	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file
		return copyURLs{Error: errInvalidSource(sourceURL).Trace()}
	}

	// All OK.. We can proceed. Type A
	return copyURLs{SourceContent: sourceContent, TargetContent: &client.Content{URL: *client.NewURL(targetURL)}}
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source URLs for copying.
func prepareCopyURLsTypeB(sourceURL string, targetURL string) copyURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return copyURLs{Error: err.Trace(sourceURL)}
	}
	if !sourceContent.Type.IsRegular() {
		if sourceContent.Type.IsDir() {
			return copyURLs{Error: errSourceIsDir(sourceURL).Trace()}
		}
		// Source is not a regular file.
		return copyURLs{Error: errInvalidSource(sourceURL).Trace()}
	}

	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	{
		sourceURLParse := client.NewURL(sourceURL)
		targetURLParse := client.NewURL(targetURL)
		targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
		return prepareCopyURLsTypeA(sourceURL, targetURLParse.String())
	}
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source URLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURL, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			copyURLsCh <- copyURLs{Error: errSourceNotRecursive(sourceURL).Trace()}
			return
		}

		// add `/` after trimming off `...` to emulate folders
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, err := url2Client(sourceURL)
		if err != nil {
			// Source initialization failed.
			copyURLsCh <- copyURLs{Error: err.Trace(sourceURL)}
			return
		}
		_, sourceContent, err := url2Stat(sourceURL)
		if err != nil {
			// if file not found it could be that user has provided a valid
			// prefix, populate parent content properly and set error to nil.
			// if at all the prefix doesn't exist eventually 'List' will handle
			// it properly.
			if os.IsNotExist(err.ToGoError()) {
				// set the err back to nil consciously.
				err = nil
				// fill sourceContent with source client data.
				sourceContent = new(client.Content)
				sourceContent.URL = sourceClient.GetURL()
				sourceContent.Type = os.ModeDir
			} else {
				// Source does not exist or insufficient privileges.
				copyURLsCh <- copyURLs{Error: err.Trace(sourceURL)}
				return
			}
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			copyURLsCh <- copyURLs{Error: errSourceIsNotDir(sourceURL).Trace()}
			return
		}

		for sourceContent := range sourceClient.List(true, false) {
			if sourceContent.Err != nil {
				// Listing failed.
				copyURLsCh <- copyURLs{Error: sourceContent.Err.Trace()}
				continue
			}

			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
			newTargetURL := client.NewURL(targetURL)
			newTargetURL.Path = filepath.Join(newTargetURL.Path,
				strings.TrimPrefix(sourceContent.Content.URL.Path, url2Dir(sourceURL)))

			copyURLsCh <- prepareCopyURLsTypeA(sourceContent.Content.URL.String(), newTargetURL.String())
		}
	}(sourceURL, targetURL, copyURLsCh)
	return copyURLsCh
}

// MULTI-SOURCE - Type D: copy([]f, d) -> []B
// prepareCopyURLsTypeD - prepares target and source URLs for copying.
func prepareCopyURLsTypeD(sourceURLs []string, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURLs []string, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)

		if sourceURLs == nil {
			// Source list is empty.
			copyURLsCh <- copyURLs{Error: errSourceListEmpty().Trace()}
			return
		}

		for _, sourceURL := range sourceURLs {
			// Target is folder. Possibilities are only Type B and C
			// Is it a recursive URL "..."?
			if isURLRecursive(sourceURL) {
				for cURLs := range prepareCopyURLsTypeC(sourceURL, targetURL) {
					copyURLsCh <- cURLs
				}
			} else {
				copyURLsCh <- prepareCopyURLsTypeB(sourceURL, targetURL)
			}
		}
	}(sourceURLs, targetURL, copyURLsCh)
	return copyURLsCh
}

// prepareCopyURLs - prepares target and source URLs for copying.
func prepareCopyURLs(sourceURLs []string, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURLs []string, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		switch guessCopyURLType(sourceURLs, targetURL) {
		case copyURLsTypeA:
			copyURLsCh <- prepareCopyURLsTypeA(sourceURLs[0], targetURL)
		case copyURLsTypeB:
			copyURLsCh <- prepareCopyURLsTypeB(sourceURLs[0], targetURL)
		case copyURLsTypeC:
			for cURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeD:
			for cURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL) {
				copyURLsCh <- cURLs
			}
		default:
			copyURLsCh <- copyURLs{Error: errInvalidArgument().Trace()}
		}
	}(sourceURLs, targetURL, copyURLsCh)

	return copyURLsCh
}
