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
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/probe"
)

type copyURLsType uint8

//   NOTE: All the parse rules should reduced to A: Copy(Source, Target).
//
//   * VALID RULES
//   =======================
//   A: copy(f, f) -> copy(f, f)
//   B: copy(f, d) -> copy(f, d/f) -> []A
//   C: copy(d1..., d2) -> []copy(f, d2/d1/f) -> []A
//   D: copy([]f, d) -> []B

//   * INVALID RULES
//   =========================
//   copy(d, f)
//   copy(d..., f)
//   copy([](f|d)..., f)

const (
	copyURLsTypeInvalid copyURLsType = iota
	copyURLsTypeA
	copyURLsTypeB
	copyURLsTypeC
	copyURLsTypeD
)

// guessCopyURLType guesses the type of clientURL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessCopyURLType(sourceURLs []string, targetURL string, isRecursive bool, keys map[string][]prefixSSEPair) (copyURLsType, *probe.Error) {
	if len(sourceURLs) == 1 { // 1 Source, 1 Target
		sourceURL := sourceURLs[0]
		_, sourceContent, err := url2Stat(sourceURL, false, keys)
		if err != nil {
			return copyURLsTypeInvalid, err
		}

		// If recursion is ON, it is type C.
		// If source is a folder, it is Type C.
		if sourceContent.Type.IsDir() || isRecursive {
			return copyURLsTypeC, nil
		}

		// If target is a folder, it is Type B.
		if isAliasURLDir(targetURL, keys) {
			return copyURLsTypeB, nil
		}
		// else Type A.
		return copyURLsTypeA, nil
	}

	// Multiple source args and target is a folder. It is Type D.
	if isAliasURLDir(targetURL, keys) {
		return copyURLsTypeD, nil
	}

	return copyURLsTypeInvalid, errInvalidArgument().Trace()
}

// SINGLE SOURCE - Type A: copy(f, f) -> copy(f, f)
// prepareCopyURLsTypeA - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeA(sourceURL string, targetURL string, encKeyDB map[string][]prefixSSEPair) URLs {
	// Extract alias before fiddling with the clientURL.
	sourceAlias, _, _ := mustExpandAlias(sourceURL)
	// Find alias and expanded clientURL.
	targetAlias, targetURL, _ := mustExpandAlias(targetURL)

	_, sourceContent, err := url2Stat(sourceURL, false, encKeyDB)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return URLs{Error: err.Trace(sourceURL)}
	}
	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file
		return URLs{Error: errInvalidSource(sourceURL).Trace(sourceURL)}
	}
	if sourceContent.URL.String() == targetURL {
		// source and target can not be same
		return URLs{Error: errSourceTargetSame(sourceURL).Trace(sourceURL)}
	}

	// All OK.. We can proceed. Type A
	return makeCopyContentTypeA(sourceAlias, sourceContent, targetAlias, targetURL, encKeyDB)
}

// prepareCopyContentTypeA - makes CopyURLs content for copying.
func makeCopyContentTypeA(sourceAlias string, sourceContent *clientContent, targetAlias string, targetURL string, encKeyDB map[string][]prefixSSEPair) URLs {
	targetContent := clientContent{URL: *newClientURL(targetURL)}

	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceContent.URL.Path))
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetContent.URL.Path))

	srcSSEKey := getSSEKey(sourcePath, encKeyDB[sourceAlias])
	tgtSSEKey := getSSEKey(targetPath, encKeyDB[targetAlias])

	return URLs{
		SourceAlias:   sourceAlias,
		SourceContent: sourceContent,
		TargetAlias:   targetAlias,
		TargetContent: &targetContent,
		SrcSSEKey:     srcSSEKey,
		TgtSSEKey:     tgtSSEKey,
	}
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeB(sourceURL string, targetURL string, encKeyDB map[string][]prefixSSEPair) URLs {
	// Extract alias before fiddling with the clientURL.
	sourceAlias, _, _ := mustExpandAlias(sourceURL)
	// Find alias and expanded clientURL.
	targetAlias, targetURL, _ := mustExpandAlias(targetURL)

	_, sourceContent, err := url2Stat(sourceURL, false, encKeyDB)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return URLs{Error: err.Trace(sourceURL)}
	}

	if !sourceContent.Type.IsRegular() {
		if sourceContent.Type.IsDir() {
			return URLs{Error: errSourceIsDir(sourceURL).Trace(sourceURL)}
		}
		// Source is not a regular file.
		return URLs{Error: errInvalidSource(sourceURL).Trace(sourceURL)}
	}

	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	return makeCopyContentTypeB(sourceAlias, sourceContent, targetAlias, targetURL, encKeyDB)
}

// makeCopyContentTypeB - CopyURLs content for copying.
func makeCopyContentTypeB(sourceAlias string, sourceContent *clientContent, targetAlias string, targetURL string, encKeyDB map[string][]prefixSSEPair) URLs {
	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	targetURLParse := newClientURL(targetURL)
	targetURLParse.Path = filepath.ToSlash(filepath.Join(targetURLParse.Path, filepath.Base(sourceContent.URL.Path)))
	return makeCopyContentTypeA(sourceAlias, sourceContent, targetAlias, targetURLParse.String(), encKeyDB)
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string, isRecursive bool, encKeyDB map[string][]prefixSSEPair) <-chan URLs {
	// Extract alias before fiddling with the clientURL.
	sourceAlias, _, _ := mustExpandAlias(sourceURL)
	// Find alias and expanded clientURL.
	targetAlias, targetURL, _ := mustExpandAlias(targetURL)
	copyURLsCh := make(chan URLs)
	go func(sourceURL, targetURL string, copyURLsCh chan URLs) {
		defer close(copyURLsCh)
		sourceClient, err := newClient(sourceURL)
		if err != nil {
			// Source initialization failed.
			copyURLsCh <- URLs{Error: err.Trace(sourceURL)}
			return
		}

		isIncomplete := false
		for sourceContent := range sourceClient.List(isRecursive, isIncomplete, DirNone) {
			if sourceContent.Err != nil {
				// Listing failed.
				copyURLsCh <- URLs{Error: sourceContent.Err.Trace(sourceClient.GetURL().String())}
				continue
			}

			if !sourceContent.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
			copyURLsCh <- makeCopyContentTypeC(sourceAlias, sourceClient.GetURL(), sourceContent, targetAlias, targetURL, encKeyDB)
		}
	}(sourceURL, targetURL, copyURLsCh)
	return copyURLsCh
}

// makeCopyContentTypeC - CopyURLs content for copying.
func makeCopyContentTypeC(sourceAlias string, sourceURL clientURL, sourceContent *clientContent, targetAlias string, targetURL string, encKeyDB map[string][]prefixSSEPair) URLs {
	newSourceURL := sourceContent.URL
	pathSeparatorIndex := strings.LastIndex(sourceURL.Path, string(sourceURL.Separator))
	newSourceSuffix := filepath.ToSlash(newSourceURL.Path)
	if pathSeparatorIndex > 1 {
		sourcePrefix := filepath.ToSlash(sourceURL.Path[:pathSeparatorIndex])
		newSourceSuffix = strings.TrimPrefix(newSourceSuffix, sourcePrefix)
	}
	newTargetURL := urlJoinPath(targetURL, newSourceSuffix)
	return makeCopyContentTypeA(sourceAlias, sourceContent, targetAlias, newTargetURL, encKeyDB)
}

// MULTI-SOURCE - Type D: copy([](f|d...), d) -> []B
// prepareCopyURLsTypeE - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeD(sourceURLs []string, targetURL string, isRecursive bool, encKeyDB map[string][]prefixSSEPair) <-chan URLs {
	copyURLsCh := make(chan URLs)
	go func(sourceURLs []string, targetURL string, copyURLsCh chan URLs) {
		defer close(copyURLsCh)
		for _, sourceURL := range sourceURLs {
			for cpURLs := range prepareCopyURLsTypeC(sourceURL, targetURL, isRecursive, encKeyDB) {
				copyURLsCh <- cpURLs
			}
		}
	}(sourceURLs, targetURL, copyURLsCh)
	return copyURLsCh
}

// prepareCopyURLs - prepares target and source clientURLs for copying.
func prepareCopyURLs(sourceURLs []string, targetURL string, isRecursive bool, encKeyDB map[string][]prefixSSEPair) <-chan URLs {
	copyURLsCh := make(chan URLs)
	go func(sourceURLs []string, targetURL string, copyURLsCh chan URLs, encKeyDB map[string][]prefixSSEPair) {
		defer close(copyURLsCh)
		cpType, err := guessCopyURLType(sourceURLs, targetURL, isRecursive, encKeyDB)
		fatalIf(err.Trace(), "Unable to guess the type of copy operation.")

		switch cpType {
		case copyURLsTypeA:
			copyURLsCh <- prepareCopyURLsTypeA(sourceURLs[0], targetURL, encKeyDB)
		case copyURLsTypeB:
			copyURLsCh <- prepareCopyURLsTypeB(sourceURLs[0], targetURL, encKeyDB)
		case copyURLsTypeC:
			for cURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL, isRecursive, encKeyDB) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeD:
			for cURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL, isRecursive, encKeyDB) {
				copyURLsCh <- cURLs
			}
		default:
			copyURLsCh <- URLs{Error: errInvalidArgument().Trace(sourceURLs...)}
		}
	}(sourceURLs, targetURL, copyURLsCh, encKeyDB)

	return copyURLsCh
}
