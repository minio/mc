// Copyright (c) 2015-2024 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"path/filepath"
	"strings"
	"time"

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
func guessCopyURLType(ctx context.Context, o prepareCopyURLsOpts) (*copyURLsContent, *probe.Error) {
	cc := new(copyURLsContent)

	// Extract alias before fiddling with the clientURL.
	cc.sourceURL = o.sourceURLs[0]
	cc.sourceAlias, _, _ = mustExpandAlias(cc.sourceURL)
	// Find alias and expanded clientURL.
	cc.targetAlias, cc.targetURL, _ = mustExpandAlias(o.targetURL)

	if len(o.sourceURLs) == 1 { // 1 Source, 1 Target
		var err *probe.Error
		if !o.isRecursive {
			_, cc.sourceContent, err = url2Stat(ctx, url2StatOptions{urlStr: cc.sourceURL, versionID: o.versionID, fileAttr: false, encKeyDB: o.encKeyDB, timeRef: o.timeRef, isZip: o.isZip, ignoreBucketExistsCheck: false})
		} else {
			_, cc.sourceContent, err = firstURL2Stat(ctx, cc.sourceURL, o.timeRef, o.isZip)
		}

		if err != nil {
			cc.copyType = copyURLsTypeInvalid
			return cc, err
		}

		// If recursion is ON, it is type C.
		// If source is a folder, it is Type C.
		if cc.sourceContent.Type.IsDir() || o.isRecursive {
			cc.copyType = copyURLsTypeC
			return cc, nil
		}

		// If target is a folder, it is Type B.
		var isDir bool
		isDir, cc.targetContent = isAliasURLDir(ctx, o.targetURL, o.encKeyDB, o.timeRef, o.ignoreBucketExistsCheck)
		if isDir {
			cc.copyType = copyURLsTypeB
			cc.sourceVersionID = cc.sourceContent.VersionID
			return cc, nil
		}

		// else Type A.
		cc.copyType = copyURLsTypeA
		cc.sourceVersionID = cc.sourceContent.VersionID
		return cc, nil
	}

	var isDir bool
	// Multiple source args and target is a folder. It is Type D.
	isDir, cc.targetContent = isAliasURLDir(ctx, o.targetURL, o.encKeyDB, o.timeRef, o.ignoreBucketExistsCheck)
	if isDir {
		cc.copyType = copyURLsTypeD
		return cc, nil
	}

	cc.copyType = copyURLsTypeInvalid
	return cc, errInvalidArgument().Trace()
}

// SINGLE SOURCE - Type A: copy(f, f) -> copy(f, f)
// prepareCopyURLsTypeA - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeA(ctx context.Context, cc copyURLsContent, o prepareCopyURLsOpts) URLs {
	var err *probe.Error
	if cc.sourceContent == nil {
		_, cc.sourceContent, err = url2Stat(ctx, url2StatOptions{urlStr: cc.sourceURL, versionID: cc.sourceVersionID, fileAttr: false, encKeyDB: o.encKeyDB, timeRef: time.Time{}, isZip: o.isZip, ignoreBucketExistsCheck: false})
		if err != nil {
			// Source does not exist or insufficient privileges.
			return URLs{Error: err.Trace(cc.sourceURL)}
		}
	}

	if !cc.sourceContent.Type.IsRegular() {
		// Source is not a regular file
		return URLs{Error: errInvalidSource(cc.sourceURL).Trace(cc.sourceURL)}
	}
	// All OK.. We can proceed. Type A
	return makeCopyContentTypeA(cc)
}

// prepareCopyContentTypeA - makes CopyURLs content for copying.
func makeCopyContentTypeA(cc copyURLsContent) URLs {
	targetContent := ClientContent{URL: *newClientURL(cc.targetURL)}
	return URLs{
		SourceAlias:   cc.sourceAlias,
		SourceContent: cc.sourceContent,
		TargetAlias:   cc.targetAlias,
		TargetContent: &targetContent,
	}
}

// SINGLE SOURCE - Type B: copy(f, d) -> copy(f, d/f) -> A
// prepareCopyURLsTypeB - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeB(ctx context.Context, cc copyURLsContent, o prepareCopyURLsOpts) URLs {
	var err *probe.Error
	if cc.sourceContent == nil {
		_, cc.sourceContent, err = url2Stat(ctx, url2StatOptions{urlStr: cc.sourceURL, versionID: cc.sourceVersionID, fileAttr: false, encKeyDB: o.encKeyDB, timeRef: time.Time{}, isZip: o.isZip, ignoreBucketExistsCheck: o.ignoreBucketExistsCheck})
		if err != nil {
			// Source does not exist or insufficient privileges.
			return URLs{Error: err.Trace(cc.sourceURL)}
		}
	}

	if !cc.sourceContent.Type.IsRegular() {
		if cc.sourceContent.Type.IsDir() {
			return URLs{Error: errSourceIsDir(cc.sourceURL).Trace(cc.sourceURL)}
		}
		// Source is not a regular file.
		return URLs{Error: errInvalidSource(cc.sourceURL).Trace(cc.sourceURL)}
	}

	if cc.targetContent == nil {
		_, cc.targetContent, err = url2Stat(ctx, url2StatOptions{urlStr: cc.targetURL, versionID: "", fileAttr: false, encKeyDB: o.encKeyDB, timeRef: time.Time{}, isZip: false, ignoreBucketExistsCheck: o.ignoreBucketExistsCheck})
		if err == nil {
			if !cc.targetContent.Type.IsDir() {
				return URLs{Error: errInvalidTarget(cc.targetURL).Trace(cc.targetURL)}
			}
		}
	}
	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	return makeCopyContentTypeB(cc)
}

// makeCopyContentTypeB - CopyURLs content for copying.
func makeCopyContentTypeB(cc copyURLsContent) URLs {
	// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
	targetURLParse := newClientURL(cc.targetURL)
	targetURLParse.Path = filepath.ToSlash(filepath.Join(targetURLParse.Path, filepath.Base(cc.sourceContent.URL.Path)))
	cc.targetURL = targetURLParse.String()
	return makeCopyContentTypeA(cc)
}

// SINGLE SOURCE - Type C: copy(d1..., d2) -> []copy(d1/f, d1/d2/f) -> []A
// prepareCopyRecursiveURLTypeC - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeC(ctx context.Context, cc copyURLsContent, o prepareCopyURLsOpts) <-chan URLs {
	copyURLsCh := make(chan URLs, 1)

	returnErrorAndCloseChannel := func(err *probe.Error) chan URLs {
		copyURLsCh <- URLs{Error: err}
		close(copyURLsCh)
		return copyURLsCh
	}

	c, err := newClient(cc.sourceURL)
	if err != nil {
		return returnErrorAndCloseChannel(err.Trace(cc.sourceURL))
	}

	if cc.targetContent == nil {
		_, cc.targetContent, err = url2Stat(ctx, url2StatOptions{urlStr: cc.targetURL, versionID: "", fileAttr: false, encKeyDB: o.encKeyDB, timeRef: time.Time{}, isZip: o.isZip, ignoreBucketExistsCheck: false})
		if err == nil {
			if !cc.targetContent.Type.IsDir() {
				return returnErrorAndCloseChannel(errTargetIsNotDir(cc.targetURL).Trace(cc.targetURL))
			}
		}
	}

	if cc.sourceContent == nil {
		_, cc.sourceContent, err = url2Stat(ctx, url2StatOptions{urlStr: cc.sourceURL, versionID: "", fileAttr: false, encKeyDB: o.encKeyDB, timeRef: time.Time{}, isZip: o.isZip, ignoreBucketExistsCheck: false})
		if err != nil {
			return returnErrorAndCloseChannel(err.Trace(cc.sourceURL))
		}
	}

	if cc.sourceContent.Type.IsDir() {
		// Require --recursive flag if we are copying a directory
		if !o.isRecursive {
			return returnErrorAndCloseChannel(errRequiresRecursive(cc.sourceURL).Trace(cc.sourceURL))
		}

		// Check if we are going to copy a directory into itself
		if isURLContains(cc.sourceURL, cc.targetURL, string(c.GetURL().Separator)) {
			return returnErrorAndCloseChannel(errCopyIntoSelf(cc.sourceURL).Trace(cc.targetURL))
		}
	}

	go func(sourceClient Client, cc copyURLsContent, o prepareCopyURLsOpts, copyURLsCh chan URLs) {
		defer close(copyURLsCh)

		for sourceContent := range sourceClient.List(ctx, ListOptions{Recursive: o.isRecursive, TimeRef: o.timeRef, ShowDir: DirNone, ListZip: o.isZip}) {
			if sourceContent.Err != nil {
				// Listing failed.
				copyURLsCh <- URLs{Error: sourceContent.Err.Trace(sourceClient.GetURL().String())}
				continue
			}

			if !sourceContent.Type.IsRegular() {
				// Source is not a regular file. Skip it for copy.
				continue
			}

			// Clone cc
			newCC := cc
			newCC.sourceContent = sourceContent
			// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
			copyURLsCh <- makeCopyContentTypeC(newCC, sourceClient.GetURL())
		}
	}(c, cc, o, copyURLsCh)

	return copyURLsCh
}

// makeCopyContentTypeC - CopyURLs content for copying.
func makeCopyContentTypeC(cc copyURLsContent, sourceClientURL ClientURL) URLs {
	newSourceURL := cc.sourceContent.URL
	pathSeparatorIndex := strings.LastIndex(sourceClientURL.Path, string(sourceClientURL.Separator))
	newSourceSuffix := filepath.ToSlash(newSourceURL.Path)
	if pathSeparatorIndex > 1 {
		sourcePrefix := filepath.ToSlash(sourceClientURL.Path[:pathSeparatorIndex])
		newSourceSuffix = strings.TrimPrefix(newSourceSuffix, sourcePrefix)
	}
	newTargetURL := urlJoinPath(cc.targetURL, newSourceSuffix)
	cc.targetURL = newTargetURL
	return makeCopyContentTypeA(cc)
}

// MULTI-SOURCE - Type D: copy([](f|d...), d) -> []B
// prepareCopyURLsTypeE - prepares target and source clientURLs for copying.
func prepareCopyURLsTypeD(ctx context.Context, cc copyURLsContent, o prepareCopyURLsOpts) <-chan URLs {
	copyURLsCh := make(chan URLs, 1)
	copyURLsFilterCh := make(chan URLs, 1)

	go func(ctx context.Context, cc copyURLsContent, o prepareCopyURLsOpts) {
		defer close(copyURLsFilterCh)

		for _, sourceURL := range o.sourceURLs {
			// Clone CC
			newCC := cc
			newCC.sourceURL = sourceURL

			for cpURLs := range prepareCopyURLsTypeC(ctx, newCC, o) {
				copyURLsFilterCh <- cpURLs
			}
		}
	}(ctx, cc, o)

	go func() {
		defer close(copyURLsCh)
		filter := make(map[string]struct{})
		for cpURLs := range copyURLsFilterCh {
			if cpURLs.Error != nil || cpURLs.TargetContent == nil {
				copyURLsCh <- cpURLs
				continue
			}

			url := cpURLs.TargetContent.URL.String()
			_, ok := filter[url]
			if !ok {
				filter[url] = struct{}{}
				copyURLsCh <- cpURLs
			}
		}
	}()

	return copyURLsCh
}

type prepareCopyURLsOpts struct {
	sourceURLs              []string
	targetURL               string
	isRecursive             bool
	encKeyDB                map[string][]prefixSSEPair
	olderThan, newerThan    string
	timeRef                 time.Time
	versionID               string
	isZip                   bool
	ignoreBucketExistsCheck bool
}

type copyURLsContent struct {
	targetContent   *ClientContent
	targetAlias     string
	targetURL       string
	sourceContent   *ClientContent
	sourceAlias     string
	sourceURL       string
	copyType        copyURLsType
	sourceVersionID string
}

// prepareCopyURLs - prepares target and source clientURLs for copying.
func prepareCopyURLs(ctx context.Context, o prepareCopyURLsOpts) chan URLs {
	copyURLsCh := make(chan URLs)
	go func(o prepareCopyURLsOpts) {
		defer close(copyURLsCh)
		copyURLsContent, err := guessCopyURLType(ctx, o)
		if err != nil {
			copyURLsCh <- URLs{Error: errUnableToGuess(err.Cause.Error()).Trace(o.sourceURLs...)}
			return
		}

		switch copyURLsContent.copyType {
		case copyURLsTypeA:
			copyURLsCh <- prepareCopyURLsTypeA(ctx, *copyURLsContent, o)
		case copyURLsTypeB:
			copyURLsCh <- prepareCopyURLsTypeB(ctx, *copyURLsContent, o)
		case copyURLsTypeC:
			for cURLs := range prepareCopyURLsTypeC(ctx, *copyURLsContent, o) {
				copyURLsCh <- cURLs
			}
		case copyURLsTypeD:
			for cURLs := range prepareCopyURLsTypeD(ctx, *copyURLsContent, o) {
				copyURLsCh <- cURLs
			}
		default:
			copyURLsCh <- URLs{Error: errInvalidArgument().Trace(o.sourceURLs...)}
		}
	}(o)

	finalCopyURLsCh := make(chan URLs)
	go func() {
		defer close(finalCopyURLsCh)
		for cpURLs := range copyURLsCh {
			if cpURLs.Error != nil {
				finalCopyURLsCh <- cpURLs
				continue
			}
			// Skip objects older than --older-than parameter if specified
			if o.olderThan != "" && isOlder(cpURLs.SourceContent.Time, o.olderThan) {
				continue
			}

			// Skip objects newer than --newer-than parameter if specified
			if o.newerThan != "" && isNewer(cpURLs.SourceContent.Time, o.newerThan) {
				continue
			}

			finalCopyURLsCh <- cpURLs
		}
	}()

	return finalCopyURLsCh
}
