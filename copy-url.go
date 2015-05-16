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
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

//
//   NOITE: All the parse rules should reduced to A: Copy(Source, Target).
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

// source2Client returns client and hostconfig objects from the source URL.
func source2Client(sourceURL string) (client.Client, error) {
	// Empty source arg?
	sourceURLParse, err := url.Parse(sourceURL)
	if err != nil || sourceURLParse.Path == "" {
		return nil, iodine.New(errInvalidSource{path: sourceURL}, nil)
	}

	sourceConfig, err := getHostConfig(sourceURL)
	if err != nil {
		return nil, iodine.New(errInvalidSource{path: sourceURL}, nil)
	}

	sourceClient, err := getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(errInvalidSource{path: sourceURL}, nil)
	}
	return sourceClient, nil
}

// target2Client returns client and hostconfig objects from the target URL.
func target2Client(targetURL string) (client.Client, error) {
	// Empty target arg?
	targetURLParse, err := url.Parse(targetURL)
	if err != nil || targetURLParse.Path == "" {
		return nil, iodine.New(errInvalidTarget{path: targetURL}, nil)
	}

	targetConfig, err := getHostConfig(targetURL)
	if err != nil {
		return nil, iodine.New(errInvalidTarget{path: targetURL}, nil)
	}

	targetClient, err := getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(errInvalidTarget{path: targetURL}, nil)
	}
	return targetClient, nil
}

// Check if the target URL represents directory. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	if strings.HasSuffix(targetURL, string(filepath.Separator)) {
		return true
	}

	targetConfig, err := getHostConfig(targetURL)
	if err != nil {
		return false
	}

	targetClient, err := getNewClient(targetURL, targetConfig, globalDebugFlag)
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
func guessCopyURLType(sourceURLs []string, targetURL string) copyURLsType {
	if targetURL == "" { // Target is empty
		return copyURLsTypeInvalid
	}

	if len(sourceURLs) < 1 { // Source list is empty
		return copyURLsTypeInvalid
	}

	if len(sourceURLs) == 1 { // 1 Source, 1 Target
		if isURLRecursive(sourceURLs[0]) { // Type C
			return copyURLsTypeC
		} // else Type A or Type B
		if isTargetURLDir(targetURL) { // Type B
			return copyURLsTypeB
		} // else Type A
		return copyURLsTypeA
	} // else Type D
	return copyURLsTypeD
}

// prepareCopyURLsTypeA - prepares target and source URLs for copying.
func prepareCopyURLsTypeA(sourceURL string, targetURL string) *copyURLs {

	sourceClient, err := source2Client(sourceURL)
	if err != nil {
		return &copyURLs{Error: iodine.New(err, nil)}
	}

	// Source exist?
	sourceContent, err := sourceClient.Stat()
	if err != nil {
		// Source does not exist or insufficient privileges.
		return &copyURLs{Error: iodine.New(errInvalidSource{path: sourceURL}, nil)}
	}
	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file
		return &copyURLs{Error: iodine.New(errInvalidSource{path: sourceURL}, nil)}
	}

	targetClient, err := target2Client(targetURL)
	if err != nil {
		return &copyURLs{Error: iodine.New(err, nil)}
	}

	targetContent, err := targetClient.Stat()
	if err == nil { // Target exists.
		if !targetContent.Type.IsRegular() { // Target is not a regular file
			return &copyURLs{Error: iodine.New(errInvalidTarget{path: targetURL}, nil)}
		}
		// Target exists but, we can may overwrite. Let the copy function decide.
	} // Target does not exist. We can create a new target file | object here.

	// All OK.. We can proceed. Type A
	return &copyURLs{SourceContent: sourceContent, TargetContent: &client.Content{Name: targetURL}}
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source URLs for copying.
func prepareCopyURLsTypeB(sourceURL string, targetURL string) *copyURLs {
	sourceClient, err := source2Client(sourceURL)
	if err != nil {
		return &copyURLs{Error: iodine.New(err, nil)}
	}

	sourceContent, err := sourceClient.Stat()
	if err != nil {
		// Source does not exist or insufficient privileges.
		return &copyURLs{Error: iodine.New(errInvalidSource{path: sourceURL}, nil)}
	}

	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file.
		return &copyURLs{Error: iodine.New(errInvalidSource{path: sourceURL}, nil)}
	}

	targetClient, err := target2Client(targetURL)
	if err != nil {
		return &copyURLs{Error: iodine.New(err, nil)}
	}

	// Target exist?
	targetContent, err := targetClient.Stat()
	/*
		if err != nil {
			// Target does not exist.
			return &copyURLs{Error: iodine.New(fmt.Errorf("Target directory [%s] does not exist.", targetURL), nil)}
		}*/

	if err == nil {
		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			return &copyURLs{Error: iodine.New(fmt.Errorf("Target [%s] is not a directory.", targetURL), nil)}
		}
	} // Else name is available to create.

	// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
	sourceURLParse, err := url.Parse(sourceURL)
	if err != nil {
		return &copyURLs{Error: iodine.New(errInvalidSource{path: sourceURL}, nil)}
	}

	targetURLParse, err := url.Parse(targetURL)
	if err != nil {
		return &copyURLs{Error: iodine.New(errInvalidTarget{path: targetURL}, nil)}
	}

	targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
	return prepareCopyURLsTypeA(sourceURL, targetURLParse.String())
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source URLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string) <-chan *copyURLs {
	copyURLsCh := make(chan *copyURLs)

	go func(sourceURL, targetURL string, copyURLsCh chan *copyURLs) {
		defer close(copyURLsCh)

		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			copyURLsCh <- &copyURLs{Error: iodine.New(fmt.Errorf("Source [%s] is not recursive.", sourceURL), nil)}
			return
		}
		sourceURL = stripRecursiveURL(sourceURL)

		sourceClient, err := source2Client(sourceURL)
		if err != nil {
			copyURLsCh <- &copyURLs{Error: iodine.New(err, nil)}
			return
		}

		// Source exist?
		sourceContent, err := sourceClient.Stat()
		if err != nil {
			// Source does not exist or insufficient privileges.
			copyURLsCh <- &copyURLs{Error: iodine.New(err, nil)}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			copyURLsCh <- &copyURLs{Error: iodine.New(fmt.Errorf("Source [%s] is not a directory.", sourceURL), nil)}
			return
		}

		targetClient, err := target2Client(targetURL)
		if err != nil {
			copyURLsCh <- &copyURLs{Error: iodine.New(err, nil)}
			return
		}

		// Target exist?
		targetContent, err := targetClient.Stat()
		if err != nil {
			// Target does not exist.
			copyURLsCh <- &copyURLs{Error: iodine.New(fmt.Errorf("Target directory [%s] does not exist.", targetURL), nil)}
			return
		}
		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			copyURLsCh <- &copyURLs{Error: iodine.New(fmt.Errorf("Target [%s] is not a directory.", targetURL), nil)}
			return
		}

		for sourceContent := range sourceClient.ListRecursive() {
			if sourceContent.Err != nil {
				// Listing failed.
				copyURLsCh <- &copyURLs{Error: iodine.New(sourceContent.Err, nil)}
				continue
			}

			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}
			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.

			sourceURLParse, err := url.Parse(sourceURL)
			if err != nil {
				copyURLsCh <- &copyURLs{Error: iodine.New(errInvalidSource{path: sourceURL}, nil)}
			}

			targetURLParse, err := url.Parse(targetURL)
			if err != nil {
				copyURLsCh <- &copyURLs{Error: iodine.New(errInvalidTarget{path: targetURL}, nil)}
			}

			// Construct target path from recursive path of source without its prefix dir.
			newTargetURLParse := *targetURLParse
			newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path,
				strings.TrimPrefix(sourceContent.Content.Name, filepath.Dir(sourceURLParse.Path)))
			copyURLsCh <- prepareCopyURLsTypeA(sourceContent.Content.Name, newTargetURLParse.String())
		}
	}(sourceURL, targetURL, copyURLsCh)

	return copyURLsCh
}

// MULTI-SOURCE - Type D: copy([]f, d) -> []B
// prepareCopyURLsTypeD - prepares target and source URLs for copying.
func prepareCopyURLsTypeD(sourceURLs []string, targetURL string) <-chan *copyURLs {
	copyURLsCh := make(chan *copyURLs)

	go func(sourceURLs []string, targetURL string, copyURLsCh chan *copyURLs) {
		defer close(copyURLsCh)

		targetClient, err := target2Client(targetURL)
		if err != nil {
			copyURLsCh <- &copyURLs{Error: iodine.New(err, nil)}
			return
		}

		// Target exist?
		targetContent, err := targetClient.Stat()
		if err != nil {
			// Target does not exist.
			copyURLsCh <- &copyURLs{Error: iodine.New(fmt.Errorf("Target directory [%s] does not exist.", targetURL), nil)}
			return
		}
		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			copyURLsCh <- &copyURLs{Error: iodine.New(fmt.Errorf("Target [%s] is not a directory.", targetURL), nil)}
			return
		}

		if len(sourceURLs) < 1 {
			// Source list is empty.
			copyURLsCh <- &copyURLs{Error: iodine.New(errors.New("Source list is empty"), nil)}
			return
		}

		for _, sourceURL := range sourceURLs {
			// Target is directory. Possibilities are only Type B and C
			// Is it a recursive URL "..."?
			if isURLRecursive(sourceURL) { // Type C
				ch := prepareCopyURLsTypeC(sourceURL, targetURL)
				for copyURLs := range ch {
					copyURLsCh <- copyURLs
				}
			} else { // Type B
				copyURLsCh <- prepareCopyURLsTypeB(sourceURL, targetURL)
			}
		}

	}(sourceURLs, targetURL, copyURLsCh)

	return copyURLsCh
}

// prepareCopyURLs - prepares target and source URLs for copying.
func prepareCopyURLs(sourceURLs []string, targetURL string) <-chan *copyURLs {
	copyURLsCh := make(chan *copyURLs)

	go func(sourceURLs []string, targetURL string, copyURLsCh chan *copyURLs) {
		defer close(copyURLsCh)
		switch guessCopyURLType(sourceURLs, targetURL) {
		case copyURLsTypeA:
			copyURLs := prepareCopyURLsTypeA(sourceURLs[0], targetURL)
			copyURLsCh <- copyURLs
		case copyURLsTypeB:
			copyURLs := prepareCopyURLsTypeB(sourceURLs[0], targetURL)
			copyURLsCh <- copyURLs
		case copyURLsTypeC:
			for copyURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL) {
				copyURLsCh <- copyURLs
			}
		case copyURLsTypeD:
			for copyURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL) {
				copyURLsCh <- copyURLs
			}
		default:
			copyURLsCh <- &copyURLs{Error: iodine.New(errors.New("Invalid arguments."), nil)}
		}
	}(sourceURLs, targetURL, copyURLsCh)

	return copyURLsCh
}
