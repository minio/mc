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

//
//   NOTE: All the parse rules should reduced to A: Copy(Source, Target).
//
//   * SINGLE SOURCE - VALID
//   =======================
//   A: copy(f, f) -> copy(f, f)
//   B: copy(f, d) -> copy(f, d/f) -> A
//   C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
//
//   * SINGLE SOURCE - INVALID
//   =========================
//   copy(d, *)
//   copy(d..., f)
//   copy(*, d...)
//
//   * MULTI-SOURCE RECURSIVE - VALID
//   ================================
//   D: copy([](d1... | f), d2) -> []copy(d1/f | f, d2/d1/f | d2/f) -> []A
//
//   * MULTI-SOURCE RECURSIVE - INVALID
//   ==================================
//   copy(*, f)

type cpURLs struct {
	SourceContent *client.Content
	TargetContent *client.Content
	Error         error
}

type cpURLsType uint8

const (
	cpURLsTypeInvalid cpURLsType = iota
	cpURLsTypeA
	cpURLsTypeB
	cpURLsTypeC
	cpURLsTypeD
)

// Check if the target URL represents directory. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return false
	}
	if strings.HasSuffix(targetURLParse.String(), string(targetURLParse.Separator)) {
		return true
	}
	targetClient, err := target2Client(targetURL)
	if err != nil {
		return false
	}
	targetContent, err := targetClient.Stat()
	if err != nil { // Cannot stat target
		return false
	}
	if !targetContent.Type.IsDir() { // Target is a dir. Type B
		return false
	}
	return true
}

// SINGLE SOURCE - Type A: copy(f, f) -> copy(f, f)
// guessCopyURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessCopyURLType(sourceURLs []string, targetURL string) cpURLsType {
	if strings.TrimSpace(targetURL) == "" || targetURL == "" { // Target is empty
		return cpURLsTypeInvalid
	}
	if sourceURLs == nil { // Source list is empty
		return cpURLsTypeInvalid
	}
	if len(sourceURLs) == 1 { // 1 Source, 1 Target
		switch {
		// Type C
		case isURLRecursive(sourceURLs[0]):
			return cpURLsTypeC
		// Type B
		case isTargetURLDir(targetURL):
			return cpURLsTypeB
		// Type A
		default:
			return cpURLsTypeA
		}
	} // else Type D
	return cpURLsTypeD
}

// prepareCopyURLsTypeA - prepares target and source URLs for copying.
func prepareCopyURLsTypeA(sourceURL string, targetURL string) <-chan cpURLs {
	cpURLsCh := make(chan cpURLs, 10000)
	go func(sourceURL, targetURL string, cpURLsCh chan cpURLs) {
		defer close(cpURLsCh)
		sourceClient, err := source2Client(sourceURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}
		// Source exists?
		sourceContent, err := sourceClient.Stat()
		if err != nil {
			// Source does not exist or insufficient privileges.
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}
		if !sourceContent.Type.IsRegular() {
			// Source is not a regular file
			cpURLsCh <- cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
			return
		}
		targetClient, err := target2Client(targetURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}
		// Target exists?
		targetContent, err := targetClient.Stat()
		if err == nil { // Target exists.
			if !targetContent.Type.IsRegular() { // Target is not a regular file
				cpURLsCh <- cpURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
				return
			}
		}
		// All OK.. We can proceed. Type A
		sourceContent.Name = sourceURL
		cpURLsCh <- cpURLs{SourceContent: sourceContent, TargetContent: &client.Content{Name: targetURL}}
	}(sourceURL, targetURL, cpURLsCh)
	return cpURLsCh
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source URLs for copying.
func prepareCopyURLsTypeB(sourceURL string, targetURL string) <-chan cpURLs {
	cpURLsCh := make(chan cpURLs, 10000)
	go func(sourceURL, targetURL string, cpURLsCh chan cpURLs) {
		defer close(cpURLsCh)
		sourceClient, err := source2Client(sourceURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}

		sourceContent, err := sourceClient.Stat()
		if err != nil {
			// Source does not exist or insufficient privileges.
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}

		if !sourceContent.Type.IsRegular() {
			// Source is not a regular file.
			cpURLsCh <- cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
			return
		}

		targetClient, err := target2Client(targetURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}

		// Target exist?
		targetContent, err := targetClient.Stat()
		if err == nil {
			if !targetContent.Type.IsDir() {
				// Target exists, but is not a directory.
				cpURLsCh <- cpURLs{Error: iodine.New(errTargetIsNotDir{URL: targetURL}, nil)}
				return
			}
		} // Else name is available to create.

		// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
		sourceURLParse, err := client.Parse(sourceURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
			return
		}

		targetURLParse, err := client.Parse(targetURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
			return
		}

		targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
		for cURLs := range prepareCopyURLsTypeA(sourceURL, targetURLParse.String()) {
			cpURLsCh <- cURLs
		}
	}(sourceURL, targetURL, cpURLsCh)
	return cpURLsCh
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source URLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string) <-chan cpURLs {
	cpURLsCh := make(chan cpURLs, 10000)
	go func(sourceURL, targetURL string, cpURLsCh chan cpURLs) {
		defer close(cpURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			cpURLsCh <- cpURLs{Error: iodine.New(errSourceNotRecursive{URL: sourceURL}, nil)}
			return
		}

		// add `/` after trimming off `...` to emulate directories
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, err := source2Client(sourceURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}

		// Source exist?
		sourceContent, err := sourceClient.Stat()
		if err != nil {
			// Source does not exist or insufficient privileges.
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			cpURLsCh <- cpURLs{Error: iodine.New(errSourceIsNotDir{URL: sourceURL}, nil)}
			return
		}

		targetClient, err := target2Client(targetURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}

		// Target exist?
		targetContent, err := targetClient.Stat()
		if err != nil {
			// Target does not exist.
			cpURLsCh <- cpURLs{Error: iodine.New(errTargetNotFound{URL: targetURL}, nil)}
			return
		}

		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			cpURLsCh <- cpURLs{Error: iodine.New(errTargetIsNotDir{URL: targetURL}, nil)}
			return
		}

		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				cpURLsCh <- cpURLs{Error: iodine.New(sourceContent.Err, nil)}
				continue
			}

			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				cpURLsCh <- cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
				continue
			}

			targetURLParse, err := client.Parse(targetURL)
			if err != nil {
				cpURLsCh <- cpURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
				continue
			}

			sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
				string(sourceURLParse.Separator))+1]
			sourceContentName := sourceContent.Content.Name
			sourceContentURL := sourceURLDelimited + sourceContentName
			sourceContentParse, err := client.Parse(sourceContentURL)
			if err != nil {
				cpURLsCh <- cpURLs{Error: iodine.New(errInvalidSource{URL: sourceContentName}, nil)}
				continue
			}

			// Construct target path from recursive path of source without its prefix dir.
			newTargetURLParse := *targetURLParse
			newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
			for cURLs := range prepareCopyURLsTypeA(sourceContentParse.String(), newTargetURLParse.String()) {
				cpURLsCh <- cURLs
			}

		}
	}(sourceURL, targetURL, cpURLsCh)
	return cpURLsCh
}

// MULTI-SOURCE - Type D: copy([]f, d) -> []B
// prepareCopyURLsTypeD - prepares target and source URLs for copying.
func prepareCopyURLsTypeD(sourceURLs []string, targetURL string) <-chan cpURLs {
	cpURLsCh := make(chan cpURLs, 10000)
	go func(sourceURLs []string, targetURL string, cpURLsCh chan cpURLs) {
		defer close(cpURLsCh)
		targetClient, err := target2Client(targetURL)
		if err != nil {
			cpURLsCh <- cpURLs{Error: iodine.New(err, nil)}
			return
		}
		// Target exist?
		targetContent, err := targetClient.Stat()
		if err != nil {
			// Target does not exist.
			cpURLsCh <- cpURLs{Error: iodine.New(errTargetNotFound{URL: targetURL}, nil)}
			return
		}
		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			cpURLsCh <- cpURLs{Error: iodine.New(errTargetIsNotDir{URL: targetURL}, nil)}
			return
		}
		if sourceURLs == nil {
			// Source list is empty.
			cpURLsCh <- cpURLs{Error: iodine.New(errSourceListEmpty{}, nil)}
			return
		}
		for _, sourceURL := range sourceURLs {
			// Target is directory. Possibilities are only Type B and C
			// Is it a recursive URL "..."?
			switch isURLRecursive(sourceURL) {
			case true:
				for cURLs := range prepareCopyURLsTypeC(sourceURL, targetURL) {
					cpURLsCh <- cURLs
				}
			case false:
				for cURLs := range prepareCopyURLsTypeB(sourceURL, targetURL) {
					cpURLsCh <- cURLs
				}
			}
		}
	}(sourceURLs, targetURL, cpURLsCh)
	return cpURLsCh
}

// prepareCopyURLs - prepares target and source URLs for copying.
func prepareCopyURLs(sourceURLs []string, targetURL string) <-chan cpURLs {
	cpURLsCh := make(chan cpURLs, 10000)
	go func(sourceURLs []string, targetURL string, cpURLsCh chan cpURLs) {
		defer close(cpURLsCh)
		switch guessCopyURLType(sourceURLs, targetURL) {
		case cpURLsTypeA:
			for cURLs := range prepareCopyURLsTypeA(sourceURLs[0], targetURL) {
				cpURLsCh <- cURLs
			}
		case cpURLsTypeB:
			for cURLs := range prepareCopyURLsTypeB(sourceURLs[0], targetURL) {
				cpURLsCh <- cURLs
			}
		case cpURLsTypeC:
			for cURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL) {
				cpURLsCh <- cURLs
			}
		case cpURLsTypeD:
			for cURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL) {
				cpURLsCh <- cURLs
			}
		default:
			cpURLsCh <- cpURLs{Error: iodine.New(errInvalidArgument{}, nil)}
		}
	}(sourceURLs, targetURL, cpURLsCh)

	return cpURLsCh
}
