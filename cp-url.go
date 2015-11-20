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
	"github.com/minio/minio-xl/pkg/probe"
)

type copyURLs struct {
	SourceContent *client.Content
	TargetContent *client.Content
	Error         *probe.Error `json:"-"`
}

type copyURLsType uint8

//   NOTE: All the parse rules should reduced to A: Copy(Source, Target).
//
//   * VALID RULES
//   =======================
//   A: copy(f, f) -> copy(f, f)
//   B: copy(f, d) -> copy(f, d/f) -> []A
//   C: copy(d1..., d2) -> []copy(f, d2/d1/f) -> []A
//   D: copy([]f, d) -> []B -> []A

//
//   * INVALID RULES
//   =========================
//   copy(d, f)
//   copy(d..., f)
//   copy([]f, f)
//'

const (
	copyURLsTypeInvalid copyURLsType = iota
	copyURLsTypeA
	copyURLsTypeB
	copyURLsTypeC
	copyURLsTypeD
)

// guessCopyURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessCopyURLType(sourceURLs []string, targetURL string, isRecursive bool) copyURLsType {
	if len(sourceURLs) == 1 { // 1 Source, 1 Target
		sourceURL := sourceURLs[0]
		_, sourceContent, err := url2Stat(sourceURL)
		if err != nil {
			return copyURLsTypeInvalid
		}
		if sourceContent.Type.IsDir() { // If source is a Dir, it is Type C.
			return copyURLsTypeC
		}

		switch {
		case isRecursive: // If recursion is ON, it is type C.
			return copyURLsTypeC
		case isTargetURLDir(targetURL): // If not type C and target is a dir, it is Type B
			return copyURLsTypeB
		default:
			return copyURLsTypeA // else Type A.
		}
	}

	// Multiple source args and taget is a dir. It is Type D.
	if isTargetURLDir(targetURL) {
		return copyURLsTypeD
	}

	return copyURLsTypeInvalid
}

// SINGLE SOURCE - Type A: copy(f, f) -> copy(f, f)
// prepareCopyURLsTypeA - prepares target and source URLs for copying.
func prepareCopyURLsTypeA(sourceURL string, targetURL string) copyURLs {
	if sourceURL == targetURL {
		// source and target can not be same
		return copyURLs{Error: errSourceTargetSame(sourceURL).Trace(sourceURL)}
	}
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return copyURLs{Error: err.Trace(sourceURL)}
	}
	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file
		return copyURLs{Error: errInvalidSource(sourceURL).Trace(sourceURL)}
	}
	// All OK.. We can proceed. Type A
	return makeCopyContentTypeA(sourceContent, targetURL)
}

// prepareCopyContentTypeA - makes CopyURLs content for copying.
func makeCopyContentTypeA(sourceContent *client.Content, targetURL string) copyURLs {
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
			return copyURLs{Error: errSourceIsDir(sourceURL).Trace(sourceURL)}
		}
		// Source is not a regular file.
		return copyURLs{Error: errInvalidSource(sourceURL).Trace(sourceURL)}
	}

	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	return makeCopyContentTypeB(sourceContent, targetURL)
}

// makeCopyContentTypeB - CopyURLs content for copying.
func makeCopyContentTypeB(sourceContent *client.Content, targetURL string) copyURLs {
	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	targetURLParse := client.NewURL(targetURL)
	targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceContent.URL.Path))
	return makeCopyContentTypeA(sourceContent, targetURLParse.String())
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source URLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string, isRecursive bool) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURL, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		sourceClient, err := url2Client(sourceURL)
		if err != nil {
			// Source initialization failed.
			copyURLsCh <- copyURLs{Error: err.Trace(sourceURL)}
			return
		}

		for sourceContent := range sourceClient.List(isRecursive, false) {
			if sourceContent.Err != nil {
				// Listing failed.
				copyURLsCh <- copyURLs{Error: sourceContent.Err.Trace(sourceClient.GetURL().String())}
				continue
			}

			if !sourceContent.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
			copyURLsCh <- makeCopyContentTypeC(sourceClient.GetURL(), sourceContent, targetURL)
		}
	}(sourceURL, targetURL, copyURLsCh)
	return copyURLsCh
}

// makeCopyContentTypeC - CopyURLs content for copying.
func makeCopyContentTypeC(sourceURL client.URL, sourceContent *client.Content, targetURL string) copyURLs {
	newSourceURL := sourceContent.URL
	pathSeparatorIndex := strings.LastIndex(sourceURL.Path, string(sourceURL.Separator))
	newSourceSuffix := newSourceURL.Path
	if pathSeparatorIndex > 1 {
		newSourceSuffix = strings.TrimPrefix(newSourceURL.Path, sourceURL.Path[:pathSeparatorIndex])
	}
	newTargetURL := urlJoinPath(targetURL, newSourceSuffix)
	return makeCopyContentTypeA(sourceContent, newTargetURL)
}

// MULTI-SOURCE - Type D: copy([]f, d) -> []B
// prepareCopyURLsTypeD - prepares target and source URLs for copying.
func prepareCopyURLsTypeD(sourceURLs []string, targetURL string) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURLs []string, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		for _, sourceURL := range sourceURLs {
			copyURLsCh <- prepareCopyURLsTypeB(sourceURL, targetURL)
		}
	}(sourceURLs, targetURL, copyURLsCh)
	return copyURLsCh
}

// prepareCopyURLs - prepares target and source URLs for copying.
func prepareCopyURLs(sourceURLs []string, targetURL string, isRecursive bool) <-chan copyURLs {
	copyURLsCh := make(chan copyURLs)
	go func(sourceURLs []string, targetURL string, copyURLsCh chan copyURLs) {
		defer close(copyURLsCh)
		switch guessCopyURLType(sourceURLs, targetURL, isRecursive) {
		case copyURLsTypeA:
			copyURLsCh <- prepareCopyURLsTypeA(sourceURLs[0], targetURL)
		case copyURLsTypeB:
			copyURLsCh <- prepareCopyURLsTypeB(sourceURLs[0], targetURL)
		case copyURLsTypeC:
			for cURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL, isRecursive) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeD:
			for cURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL) {
				copyURLsCh <- cURLs
			}
		default:
			copyURLsCh <- copyURLs{Error: errInvalidArgument().Trace(sourceURLs...)}
		}
	}(sourceURLs, targetURL, copyURLsCh)

	return copyURLsCh
}
