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
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

type copyURLs struct {
	SourceContent *client.Content
	TargetContent *client.Content
	Error         error
}

type copyURLsType uint8

const (
	copyURLsTypeInvalid copyURLsType = iota
	copyURLsTypeA
	copyURLsTypeB
	copyURLsTypeC
	copyURLsTypeD
)

//   NOTE: All the parse rules should reduced to A: Copy(Source, Target).
//
//   * VALID RULES
//   =======================
//   A: copy(f, f) -> copy(f, f)
//   B: copy(f, d) -> copy(f, d/f) -> A
//   C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
//
//   * INVALID RULES
//   =========================
//   A: copy(d, *)
//   B: copy(d..., f)
//   C: copy(*, d...)
//

// guessCopyURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessCopyURLType(sourceURLs []string, targetURL string) copyURLsType {
	if strings.TrimSpace(targetURL) == "" || targetURL == "" { // Target is empty
		return copyURLsTypeInvalid
	}
	if sourceURLs == nil { // Source list is empty
		return copyURLsTypeInvalid
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
func prepareCopyURLsTypeA(sourceURL string, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURL, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		_, sourceContent, err := url2Stat(sourceURL)
		if err != nil {
			// Source does not exist or insufficient privileges.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}
		if !sourceContent.Type.IsRegular() {
			// Source is not a regular file
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
			return
		}
		/* Too expensive. Ignore this for now. Let it fail while putObject.
		targetClient, err := target2Client(targetURL)
		if err != nil {
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}

		// Target exists?
		targetContent, err := targetClient.Stat()
		if err == nil { // Target exists.
			if !targetContent.Type.IsRegular() { // Target is not a regular file
				copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
				return
			}
			copyURLsCh <- copyURLs{SourceContent: sourceContent, TargetContent: targetContent}
			return
		}
		*/
		// All OK.. We can proceed. Type A
		sourceContent.Name = sourceURL
		copyURLsCh <- copyURLs{SourceContent: sourceContent, TargetContent: &client.Content{Name: targetURL}}
	}(sourceURL, targetURL, copyURLsCh)
	return copyURLsCh
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source URLs for copying.
func prepareCopyURLsTypeB(sourceURL string, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURL, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		_, sourceContent, err := url2Stat(sourceURL)
		if err != nil {
			// Source does not exist or insufficient privileges.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}

		if !sourceContent.Type.IsRegular() {
			// Source is not a regular file.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
			return
		}

		_, targetContent, err := url2Stat(targetURL)
		if err != nil {
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}
		if err == nil {
			if !targetContent.Type.IsDir() {
				// Target exists, but is not a directory.
				copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errTargetIsNotDir{URL: targetURL}, nil))}
				return
			}
		} // Else name is available to create.

		// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
		sourceURLParse, err := client.Parse(sourceURL)
		if err != nil {
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
			return
		}

		targetURLParse, err := client.Parse(targetURL)
		if err != nil {
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
			return
		}

		targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
		for cURLs := range prepareCopyURLsTypeA(sourceURL, targetURLParse.String()) {
			copyURLsCh <- cURLs
		}
	}(sourceURL, targetURL, copyURLsCh)
	return copyURLsCh
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source URLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURL, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errSourceNotRecursive{URL: sourceURL}, nil))}
			return
		}

		// add `/` after trimming off `...` to emulate directories
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, sourceContent, err := url2Stat(sourceURL)
		if err != nil {
			// Source does not exist or insufficient privileges.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errSourceIsNotDir{URL: sourceURL}, nil))}
			return
		}

		_, targetContent, err := url2Stat(targetURL)
		// Target exist?
		if err != nil {
			// Target does not exist.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errTargetNotFound{URL: targetURL}, nil))}
			return
		}

		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errTargetIsNotDir{URL: targetURL}, nil))}
			return
		}

		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(sourceContent.Err, nil))}
				continue
			}

			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
				continue
			}

			targetURLParse, err := client.Parse(targetURL)
			if err != nil {
				copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
				continue
			}

			sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
				string(sourceURLParse.Separator))+1]
			sourceContentName := sourceContent.Content.Name
			sourceContentURL := sourceURLDelimited + sourceContentName
			sourceContentParse, err := client.Parse(sourceContentURL)
			if err != nil {
				copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceContentName}, nil))}
				continue
			}

			// Construct target path from recursive path of source without its prefix dir.
			newTargetURLParse := *targetURLParse
			newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
			for cURLs := range prepareCopyURLsTypeA(sourceContentParse.String(), newTargetURLParse.String()) {
				copyURLsCh <- cURLs
			}

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
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errSourceListEmpty{}, nil))}
			return
		}

		_, targetContent, err := url2Stat(targetURL)
		// Target exist?
		if err != nil {
			// Target does not exist.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errTargetNotFound{URL: targetURL}, nil))}
			return
		}

		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errTargetIsNotDir{URL: targetURL}, nil))}
			return
		}

		for _, sourceURL := range sourceURLs {
			// Target is directory. Possibilities are only Type B and C
			// Is it a recursive URL "..."?
			switch isURLRecursive(sourceURL) {
			case true:
				for cURLs := range prepareCopyURLsTypeC(sourceURL, targetURL) {
					copyURLsCh <- cURLs
				}
			case false:
				for cURLs := range prepareCopyURLsTypeB(sourceURL, targetURL) {
					copyURLsCh <- cURLs
				}
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
			for cURLs := range prepareCopyURLsTypeA(sourceURLs[0], targetURL) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeB:
			for cURLs := range prepareCopyURLsTypeB(sourceURLs[0], targetURL) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeC:
			for cURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeD:
			for cURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL) {
				copyURLsCh <- cURLs
			}
		default:
			copyURLsCh <- copyURLs{Error: NewIodine(iodine.New(errInvalidArgument{}, nil))}
		}
	}(sourceURLs, targetURL, copyURLsCh)

	return copyURLsCh
}
