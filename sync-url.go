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
//   A: sync(f, f) -> copy(f, f)
//   B: sync(f, d) -> copy(f, d/f) -> A
//   C: sync(f, []d) -> []copy(f, d/f) -> []A
//      sync(d1..., d2) -> []copy(d1/f, d2/d1/f) -> []A
//      sync(d1..., []d2) -> [][]copy(d1/f, d2/d1/f) -> [][]A
//
//   * SINGLE SOURCE - INVALID
//   =========================
//   sync(d, *)
//   sync(d..., f)
//   sync(*, d...)
//
//   * MULTI-TARGET RECURSIVE - INVALID
//   ==================================
//   sync(*, f1)
//   sync(*, []f1)

type syncURLs struct {
	SourceContent  *client.Content
	TargetContents []*client.Content
	Error          error
}

func (s syncURLs) IsEmpty() bool {
	empty := false
	if s.SourceContent == nil {
		empty = true
		if s.TargetContents == nil {
			empty = true
			return empty
		}
		if len(s.TargetContents) > 0 && s.TargetContents[0] == nil {
			empty = true
			return empty
		}
	}
	return empty
}

type syncURLsType cpURLsType

// guessSyncURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessSyncURLType(sourceURL string, targetURLs []string) syncURLsType {
	if targetURLs == nil { // Target is empty
		return syncURLsType(cpURLsTypeInvalid)
	}
	if sourceURL == "" { // Source list is empty
		return syncURLsType(cpURLsTypeInvalid)
	}
	if isURLRecursive(sourceURL) { // Type C
		return syncURLsType(cpURLsTypeC)
	} // else Type A or Type B
	for _, targetURL := range targetURLs {
		if isTargetURLDir(targetURL) { // Type B
			return syncURLsType(cpURLsTypeB)
		}
	} // else Type A
	return syncURLsType(cpURLsTypeA)
}

// prepareSyncURLsTypeA - A: sync(f, f) -> copy(f, f)
func prepareSyncURLsTypeA(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs, 10000)
	go func() {
		defer close(syncURLsCh)
		var sURLs syncURLs
		for _, targetURL := range targetURLs {
			var cURLs cpURLs
			for cURLs = range prepareCopyURLsTypeA(sourceURL, targetURL) {
				if cURLs.Error != nil {
					syncURLsCh <- syncURLs{Error: iodine.New(cURLs.Error, nil)}
					continue
				}
			}
			sURLs.SourceContent = cURLs.SourceContent
			sURLs.TargetContents = append(sURLs.TargetContents, cURLs.TargetContent)
		}
		if !sURLs.IsEmpty() {
			syncURLsCh <- sURLs
		}
	}()
	return syncURLsCh
}

// prepareSyncURLsTypeB - B: sync(f, d) -> copy(f, d/f) -> A
func prepareSyncURLsTypeB(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs, 10000)
	go func() {
		defer close(syncURLsCh)
		var sURLs syncURLs
		for _, targetURL := range targetURLs {
			var cURLs cpURLs
			for cURLs = range prepareCopyURLsTypeB(sourceURL, targetURL) {
				if cURLs.Error != nil {
					syncURLsCh <- syncURLs{Error: iodine.New(cURLs.Error, nil)}
					continue
				}
			}
			sURLs.SourceContent = cURLs.SourceContent
			sURLs.TargetContents = append(sURLs.TargetContents, cURLs.TargetContent)
		}
		if !sURLs.IsEmpty() {
			syncURLsCh <- sURLs
		}
	}()
	return syncURLsCh
}

// prepareSyncURLsTypeC - C: sync(f, []d) -> []copy(f, d/f) -> []A
func prepareSyncURLsTypeC(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs, 10000)
	go func() {
		defer close(syncURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			syncURLsCh <- syncURLs{Error: iodine.New(errSourceNotRecursive{URL: sourceURL}, nil)}
			return
		}
		// add `/` after trimming off `...` to emulate directories
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, sourceContent, err := url2Stat(sourceURL)
		// Source exist?
		if err != nil {
			// Source does not exist or insufficient privileges.
			syncURLsCh <- syncURLs{Error: iodine.New(err, nil)}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			syncURLsCh <- syncURLs{Error: iodine.New(errSourceIsNotDir{URL: sourceURL}, nil)}
			return
		}

		for _, targetURL := range targetURLs {
			_, targetContent, err := url2Stat(targetURL)
			// Target exist?
			if err != nil {
				// Target does not exist.
				syncURLsCh <- syncURLs{Error: iodine.New(errTargetNotFound{URL: targetURL}, nil)}
				return
			}

			if !targetContent.Type.IsDir() {
				// Target exists, but is not a directory.
				syncURLsCh <- syncURLs{Error: iodine.New(errTargetIsNotDir{URL: targetURL}, nil)}
				return
			}
		}
		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				syncURLsCh <- syncURLs{Error: iodine.New(sourceContent.Err, nil)}
				continue
			}
			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}
			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				syncURLsCh <- syncURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
				continue
			}
			var newTargetURLs []string
			var sourceContentParse *client.URL
			for _, targetURL := range targetURLs {
				targetURLParse, err := client.Parse(targetURL)
				if err != nil {
					syncURLsCh <- syncURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
					continue
				}
				sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
					string(sourceURLParse.Separator))+1]
				sourceContentName := sourceContent.Content.Name
				sourceContentURL := sourceURLDelimited + sourceContentName
				sourceContentParse, err = client.Parse(sourceContentURL)
				if err != nil {
					syncURLsCh <- syncURLs{Error: iodine.New(errInvalidSource{URL: sourceContentName}, nil)}
					continue
				}
				// Construct target path from recursive path of source without its prefix dir.
				newTargetURLParse := *targetURLParse
				newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
				newTargetURLs = append(newTargetURLs, newTargetURLParse.String())
			}
			for sURLs := range prepareSyncURLsTypeA(sourceContentParse.String(), newTargetURLs) {
				syncURLsCh <- sURLs
			}
		}
	}()
	return syncURLsCh
}

// prepareCopyURLs - prepares target and source URLs for syncing.
func prepareSyncURLs(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs, 10000)
	go func() {
		defer close(syncURLsCh)
		switch guessSyncURLType(sourceURL, targetURLs) {
		case syncURLsType(cpURLsTypeA):
			for sURLs := range prepareSyncURLsTypeA(sourceURL, targetURLs) {
				syncURLsCh <- sURLs
			}
		case syncURLsType(cpURLsTypeB):
			for sURLs := range prepareSyncURLsTypeB(sourceURL, targetURLs) {
				syncURLsCh <- sURLs
			}
		case syncURLsType(cpURLsTypeC):
			for sURLs := range prepareSyncURLsTypeC(sourceURL, targetURLs) {
				syncURLsCh <- sURLs
			}
		default:
			syncURLsCh <- syncURLs{Error: iodine.New(errInvalidArgument{}, nil)}
		}
	}()
	return syncURLsCh
}
