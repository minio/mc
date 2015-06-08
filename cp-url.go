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
	if strings.HasSuffix(targetURL, string(filepath.Separator)) {
		return true
	}

	targetConfig, err := getHostConfig(targetURL)
	if err != nil {
		return false
	}

	targetClient, err := getNewClient(targetURL, targetConfig)
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
	if targetURL == "" { // Target is empty
		return cpURLsTypeInvalid
	}

	if len(sourceURLs) < 1 { // Source list is empty
		return cpURLsTypeInvalid
	}

	if len(sourceURLs) == 1 { // 1 Source, 1 Target
		if isURLRecursive(sourceURLs[0]) { // Type C
			return cpURLsTypeC
		} // else Type A or Type B
		if isTargetURLDir(targetURL) { // Type B
			return cpURLsTypeB
		} // else Type A
		return cpURLsTypeA
	} // else Type D
	return cpURLsTypeD
}

// prepareCopyURLsTypeA - prepares target and source URLs for copying.
func prepareCopyURLsTypeA(sourceURL string, targetURL string) *cpURLs {
	sourceClient, err := source2Client(sourceURL)
	if err != nil {
		return &cpURLs{Error: iodine.New(err, nil)}
	}

	// Source exists?
	sourceContent, err := sourceClient.Stat()
	if err != nil {
		// Source does not exist or insufficient privileges.
		return &cpURLs{Error: iodine.New(err, nil)}
	}
	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file
		return &cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
	}

	targetClient, err := target2Client(targetURL)
	if err != nil {
		return &cpURLs{Error: iodine.New(err, nil)}
	}
	// Target exists?
	targetContent, err := targetClient.Stat()
	if err == nil { // Target exists.
		if !targetContent.Type.IsRegular() { // Target is not a regular file
			return &cpURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
		}
	}

	// All OK.. We can proceed. Type A
	sourceContent.Name = sourceURL
	return &cpURLs{SourceContent: sourceContent, TargetContent: &client.Content{Name: targetURL}}
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source URLs for copying.
func prepareCopyURLsTypeB(sourceURL string, targetURL string) *cpURLs {
	sourceClient, err := source2Client(sourceURL)
	if err != nil {
		return &cpURLs{Error: iodine.New(err, nil)}
	}

	sourceContent, err := sourceClient.Stat()
	if err != nil {
		// Source does not exist or insufficient privileges.
		return &cpURLs{Error: iodine.New(err, nil)}
	}

	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file.
		return &cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
	}

	targetClient, err := target2Client(targetURL)
	if err != nil {
		return &cpURLs{Error: iodine.New(err, nil)}
	}

	// Target exist?
	targetContent, err := targetClient.Stat()
	if err == nil {
		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			return &cpURLs{Error: iodine.New(fmt.Errorf("Target [%s] is not a directory.", targetURL), nil)}
		}
	} // Else name is available to create.

	// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
	sourceURLParse, err := client.Parse(sourceURL)
	if err != nil {
		return &cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
	}

	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return &cpURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
	}

	targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
	return prepareCopyURLsTypeA(sourceURL, targetURLParse.String())
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source URLs for copying.
func prepareCopyURLsTypeC(sourceURL, targetURL string) <-chan *cpURLs {
	cpURLsCh := make(chan *cpURLs)
	go func(sourceURL, targetURL string, cpURLsCh chan *cpURLs) {
		defer close(cpURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			cpURLsCh <- &cpURLs{Error: iodine.New(fmt.Errorf("Source [%s] is not recursive.", sourceURL), nil)}
			return
		}

		// add `/` after trimming off `...` to emulate directories
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, err := source2Client(sourceURL)
		if err != nil {
			cpURLsCh <- &cpURLs{Error: iodine.New(err, nil)}
			return
		}

		// Source exist?
		sourceContent, err := sourceClient.Stat()
		if err != nil {
			// Source does not exist or insufficient privileges.
			cpURLsCh <- &cpURLs{Error: iodine.New(err, nil)}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			cpURLsCh <- &cpURLs{Error: iodine.New(fmt.Errorf("Source [%s] is not a directory.", sourceURL), nil)}
			return
		}

		targetClient, err := target2Client(targetURL)
		if err != nil {
			cpURLsCh <- &cpURLs{Error: iodine.New(err, nil)}
			return
		}

		// Target exist?
		targetContent, err := targetClient.Stat()
		if err != nil {
			// Target does not exist.
			cpURLsCh <- &cpURLs{Error: iodine.New(fmt.Errorf("Target directory [%s] does not exist.", targetURL), nil)}
			return
		}

		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			cpURLsCh <- &cpURLs{Error: iodine.New(fmt.Errorf("Target [%s] is not a directory.", targetURL), nil)}
			return
		}

		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				cpURLsCh <- &cpURLs{Error: iodine.New(sourceContent.Err, nil)}
				continue
			}

			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				cpURLsCh <- &cpURLs{Error: iodine.New(errInvalidSource{URL: sourceURL}, nil)}
				continue
			}

			targetURLParse, err := client.Parse(targetURL)
			if err != nil {
				cpURLsCh <- &cpURLs{Error: iodine.New(errInvalidTarget{URL: targetURL}, nil)}
				continue
			}

			sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
				string(sourceURLParse.Separator))+1]
			sourceContentName := sourceContent.Content.Name
			sourceContentURL := sourceURLDelimited + sourceContentName
			sourceContentParse, err := client.Parse(sourceContentURL)
			if err != nil {
				cpURLsCh <- &cpURLs{Error: iodine.New(errInvalidSource{URL: sourceContentName}, nil)}
				continue
			}

			// Construct target path from recursive path of source without its prefix dir.
			newTargetURLParse := *targetURLParse
			newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
			cpURLsCh <- prepareCopyURLsTypeA(sourceContentParse.String(), newTargetURLParse.String())
		}
	}(sourceURL, targetURL, cpURLsCh)

	return cpURLsCh
}

// MULTI-SOURCE - Type D: copy([]f, d) -> []B
// prepareCopyURLsTypeD - prepares target and source URLs for copying.
func prepareCopyURLsTypeD(sourceURLs []string, targetURL string) <-chan *cpURLs {
	cpURLsCh := make(chan *cpURLs)

	go func(sourceURLs []string, targetURL string, cpURLsCh chan *cpURLs) {
		defer close(cpURLsCh)

		targetClient, err := target2Client(targetURL)
		if err != nil {
			cpURLsCh <- &cpURLs{Error: iodine.New(err, nil)}
			return
		}

		// Target exist?
		targetContent, err := targetClient.Stat()
		if err != nil {
			// Target does not exist.
			cpURLsCh <- &cpURLs{Error: iodine.New(fmt.Errorf("Target directory [%s] does not exist.", targetURL), nil)}
			return
		}
		if !targetContent.Type.IsDir() {
			// Target exists, but is not a directory.
			cpURLsCh <- &cpURLs{Error: iodine.New(fmt.Errorf("Target [%s] is not a directory.", targetURL), nil)}
			return
		}

		if len(sourceURLs) < 1 {
			// Source list is empty.
			cpURLsCh <- &cpURLs{Error: iodine.New(errors.New("Source list is empty"), nil)}
			return
		}

		for _, sourceURL := range sourceURLs {
			// Target is directory. Possibilities are only Type B and C
			// Is it a recursive URL "..."?
			if isURLRecursive(sourceURL) { // Type C
				ch := prepareCopyURLsTypeC(sourceURL, targetURL)
				for cpURLs := range ch {
					cpURLsCh <- cpURLs
				}
			} else { // Type B
				cpURLsCh <- prepareCopyURLsTypeB(sourceURL, targetURL)
			}
		}

	}(sourceURLs, targetURL, cpURLsCh)

	return cpURLsCh
}

// prepareCopyURLs - prepares target and source URLs for copying.
func prepareCopyURLs(sourceURLs []string, targetURL string) <-chan *cpURLs {
	cpURLsCh := make(chan *cpURLs)

	go func(sourceURLs []string, targetURL string, cpURLsCh chan *cpURLs) {
		defer close(cpURLsCh)
		switch guessCopyURLType(sourceURLs, targetURL) {
		case cpURLsTypeA:
			cpURLs := prepareCopyURLsTypeA(sourceURLs[0], targetURL)
			cpURLsCh <- cpURLs
		case cpURLsTypeB:
			cpURLs := prepareCopyURLsTypeB(sourceURLs[0], targetURL)
			cpURLsCh <- cpURLs
		case cpURLsTypeC:
			for cpURLs := range prepareCopyURLsTypeC(sourceURLs[0], targetURL) {
				cpURLsCh <- cpURLs
			}
		case cpURLsTypeD:
			for cpURLs := range prepareCopyURLsTypeD(sourceURLs, targetURL) {
				cpURLsCh <- cpURLs
			}
		default:
			cpURLsCh <- &cpURLs{Error: iodine.New(errInvalidArgument{}, nil)}
		}
	}(sourceURLs, targetURL, cpURLsCh)

	return cpURLsCh
}
