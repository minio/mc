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
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

type syncURLs struct {
	SourceContent  *client.Content
	TargetContents []*client.Content
	Error          error `json:"-"`
}

type syncURLsType uint8

const (
	syncURLsTypeInvalid syncURLsType = iota
	syncURLsTypeA
	syncURLsTypeB
	syncURLsTypeC
	syncURLsTypeD
)

//   NOTE: All the parse rules should reduced to A: Sync(Source, []Target).
//
//   * SYNC ARGS - VALID CASES
//   =========================
//   A: sync(f, []f) -> sync(f, []f)
//   B: sync(f, [](d | f)) -> sync(f, [](d/f | f)) -> A:
//   C: sync(d1..., [](d2 | f)) -> []sync(d1/f, [](d1/d2/f | d1/f)) -> []A:
//
//   * SYNC ARGS - INVALID CASES
//   ===========================
//   sync(d, *)
//   sync(d..., f)
//   sync(*, d...)

// guessSyncURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessSyncURLType(sourceURL string, targetURLs []string) syncURLsType {
	if targetURLs == nil || len(targetURLs) == 0 { // Target is empty
		return syncURLsTypeInvalid
	}
	if sourceURL == "" { // Source is empty
		return syncURLsTypeInvalid
	}
	for _, targetURL := range targetURLs {
		if targetURL == "" { // One of the target is empty
			return syncURLsTypeInvalid
		}
	}

	if isURLRecursive(sourceURL) { // Type C
		return syncURLsTypeC
	} // else Type A or Type B
	for _, targetURL := range targetURLs {
		if isTargetURLDir(targetURL) { // Type B
			return syncURLsTypeB
		}
	} // else Type A
	return syncURLsTypeA
}

// prepareSingleSyncURLTypeA - prepares a single source and single target argument for syncing.
func prepareSingleSyncURLsTypeA(sourceURL string, targetURL string) syncURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil { // Source does not exist or insufficient privileges.
		return syncURLs{Error: NewIodine(iodine.New(err, nil))}
	}
	if !sourceContent.Type.IsRegular() { // Source is not a regular file
		return syncURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}
	targetClient, err := target2Client(targetURL)
	if err != nil {
		return syncURLs{Error: NewIodine(iodine.New(err, nil))}
	}
	// Target exists?
	targetContent, err := targetClient.Stat()
	if err == nil { // Target exists.
		if !targetContent.Type.IsRegular() { // Target is not a regular file
			return syncURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
		}
		var targetContents []*client.Content
		targetContents = append(targetContents, targetContent)
		return syncURLs{SourceContent: sourceContent, TargetContents: targetContents}
	}
	// All OK.. We can proceed. Type A
	sourceContent.Name = sourceURL
	return syncURLs{SourceContent: sourceContent, TargetContents: []*client.Content{{Name: targetURL}}}
}

// prepareSyncURLsTypeA - A: sync(f, f) -> sync(f, f)
func prepareSyncURLsTypeA(sourceURL string, targetURLs []string) syncURLs {
	var sURLs syncURLs
	for _, targetURL := range targetURLs { // Prepare each target separately
		URLs := prepareSingleSyncURLsTypeA(sourceURL, targetURL)
		if URLs.Error != nil {
			return syncURLs{Error: NewIodine(iodine.New(URLs.Error, nil))}
		}
		sURLs.SourceContent = URLs.SourceContent
		sURLs.TargetContents = append(sURLs.TargetContents, URLs.TargetContents...)
	}
	return sURLs
}

// prepareSingleSyncURLsTypeB - prepares a single target and single source URLs for syncing.
func prepareSingleSyncURLsTypeB(sourceURL string, targetURL string) syncURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return syncURLs{Error: NewIodine(iodine.New(err, nil))}
	}

	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file.
		return syncURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	_, targetContent, err := url2Stat(targetURL)
	if os.IsNotExist(iodine.ToError(err)) {
		// Source and target are files. Already reduced to Type A.
		return prepareSingleSyncURLsTypeA(sourceURL, targetURL)
	}
	if err != nil {
		return syncURLs{Error: NewIodine(iodine.New(err, nil))}
	}

	if targetContent.Type.IsRegular() { // File to File
		// Source and target are files. Already reduced to Type A.
		return prepareSingleSyncURLsTypeA(sourceURL, targetURL)
	}

	// Source is a file, target is a directory and exists.
	sourceURLParse, err := client.Parse(sourceURL)
	if err != nil {
		return syncURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return syncURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
	}
	// Reduce Type B to Type A.
	targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
	return prepareSingleSyncURLsTypeA(sourceURL, targetURLParse.String())
}

// prepareSyncURLsTypeB - B: sync(f, d) -> sync(f, d/f) -> A
func prepareSyncURLsTypeB(sourceURL string, targetURLs []string) syncURLs {
	var sURLs syncURLs
	for _, targetURL := range targetURLs {
		URLs := prepareSingleSyncURLsTypeB(sourceURL, targetURL)
		if URLs.Error != nil {
			return syncURLs{Error: NewIodine(iodine.New(URLs.Error, nil))}
		}
		sURLs.SourceContent = URLs.SourceContent
		sURLs.TargetContents = append(sURLs.TargetContents, URLs.TargetContents[0])
	}
	return sURLs
}

// prepareSyncURLsTypeC - C: sync(f, []d) -> []sync(f, d/f) -> []A
func prepareSyncURLsTypeC(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs)
	go func() {
		defer close(syncURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errSourceNotRecursive{URL: sourceURL}, nil))}
			return
		}
		// add `/` after trimming off `...` to emulate directories
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, sourceContent, err := url2Stat(sourceURL)
		// Source exist?
		if err != nil {
			// Source does not exist or insufficient privileges.
			syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errSourceIsNotDir{URL: sourceURL}, nil))}
			return
		}

		// Type C requires all targets to be a dir and it should exist.
		for _, targetURL := range targetURLs {
			_, targetContent, err := url2Stat(targetURL)
			// Target exist?
			if err != nil {
				// Target does not exist.
				syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errTargetNotFound{URL: targetURL}, nil))}
				return
			}

			if !targetContent.Type.IsDir() {
				// Target exists, but is not a directory.
				syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errTargetIsNotDir{URL: targetURL}, nil))}
				return
			}
		}

		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(sourceContent.Err, nil))}
				continue
			}
			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for sync.
				continue
			}
			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
				continue
			}
			var newTargetURLs []string
			var sourceContentParse *client.URL
			for _, targetURL := range targetURLs {
				targetURLParse, err := client.Parse(targetURL)
				if err != nil {
					syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
					continue
				}
				sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
					string(sourceURLParse.Separator))+1]
				sourceContentName := sourceContent.Content.Name
				sourceContentURL := sourceURLDelimited + sourceContentName
				sourceContentParse, err = client.Parse(sourceContentURL)
				if err != nil {
					syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceContentName}, nil))}
					continue
				}
				// Construct target path from recursive path of source without its prefix dir.
				newTargetURLParse := *targetURLParse
				newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
				newTargetURLs = append(newTargetURLs, newTargetURLParse.String())
			}
			syncURLsCh <- prepareSyncURLsTypeA(sourceContentParse.String(), newTargetURLs)
		}
	}()
	return syncURLsCh
}

// prepareSyncURLs - prepares target and source URLs for syncing.
func prepareSyncURLs(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs)
	go func() {
		defer close(syncURLsCh)
		switch guessSyncURLType(sourceURL, targetURLs) {
		case syncURLsType(syncURLsTypeA):
			syncURLsCh <- prepareSyncURLsTypeA(sourceURL, targetURLs)
		case syncURLsType(syncURLsTypeB):
			syncURLsCh <- prepareSyncURLsTypeB(sourceURL, targetURLs)
		case syncURLsType(syncURLsTypeC):
			for sURLs := range prepareSyncURLsTypeC(sourceURL, targetURLs) {
				syncURLsCh <- sURLs
			}
		default:
			syncURLsCh <- syncURLs{Error: NewIodine(iodine.New(errInvalidArgument{}, nil))}
		}
	}()
	return syncURLsCh
}
