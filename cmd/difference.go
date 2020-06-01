/*
 * MinIO Client (C) 2015 MinIO, Inc.
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
	"strings"
	"time"
	"unicode/utf8"

	// golang does not support flat keys for path matching, find does

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v6"
	"golang.org/x/text/unicode/norm"
)

// differType difference in type.
type differType int

const (
	differInNone          differType = iota // does not differ
	differInSize                            // differs in size
	differInMetadata                        // differs in metadata
	differInType                            // differs in type, exfile/directory
	differInFirst                           // only in source (FIRST)
	differInSecond                          // only in target (SECOND)
	differInAASourceMTime                   // differs in active-active source modtime
)

func (d differType) String() string {
	switch d {
	case differInNone:
		return ""
	case differInSize:
		return "size"
	case differInMetadata:
		return "metadata"
	case differInAASourceMTime:
		return "mm-source-mtime"
	case differInType:
		return "type"
	case differInFirst:
		return "only-in-first"
	case differInSecond:
		return "only-in-second"
	}
	return "unknown"
}

const activeActiveSourceModTimeKey = "X-Amz-Meta-Mm-Source-Mtime"

func oldestNonNullTime(t1, t2 time.Time) time.Time {
	if t1.IsZero() {
		return t2
	}
	if t2.IsZero() {
		return t1
	}
	if t1.Before(t2) {
		return t1
	}

	return t2
}

// activeActiveModTimeUpdated tries to calculate if the object copy in the target
// is older than the one in the source by comparing the modtime of the data.
func activeActiveModTimeUpdated(src, dst *ClientContent) bool {
	if src == nil || dst == nil {
		return false
	}

	if src.Time.IsZero() || dst.Time.IsZero() {
		// This should only happen in a messy environment
		// but we are returning false anyway so the caller
		// function won't take any action.
		return false
	}

	_, ok1 := src.UserMetadata[activeActiveSourceModTimeKey]
	_, ok2 := dst.UserMetadata[activeActiveSourceModTimeKey]
	if !ok1 && !ok2 {
		// No multimaster context found, consider src & dst as similar
		return false
	}

	srcOriginLastModified, _ := time.Parse(time.RFC3339Nano, src.UserMetadata[activeActiveSourceModTimeKey])
	dstOriginLastModified, _ := time.Parse(time.RFC3339Nano, dst.UserMetadata[activeActiveSourceModTimeKey])

	srcActualModTime := oldestNonNullTime(src.Time, srcOriginLastModified)
	dstActualModTime := oldestNonNullTime(dst.Time, dstOriginLastModified)

	return srcActualModTime.After(dstActualModTime)
}

func metadataEqual(m1, m2 map[string]string) bool {
	for k, v := range m1 {
		if k == activeActiveSourceModTimeKey {
			continue
		}
		if m2[k] != v {
			return false
		}
	}
	for k, v := range m2 {
		if k == activeActiveSourceModTimeKey {
			continue
		}
		if m1[k] != v {
			return false
		}
	}
	return true
}

func objectDifference(sourceClnt, targetClnt Client, sourceURL, targetURL string, isMetadata bool) (diffCh chan diffMessage) {
	return difference(sourceClnt, targetClnt, sourceURL, targetURL, isMetadata, true, false, DirNone)
}

func dirDifference(sourceClnt, targetClnt Client, sourceURL, targetURL string) (diffCh chan diffMessage) {
	return difference(sourceClnt, targetClnt, sourceURL, targetURL, false, false, true, DirFirst)
}

func differenceInternal(sourceClnt, targetClnt Client, sourceURL, targetURL string, isMetadata bool, isRecursive, returnSimilar bool, dirOpt DirOpt, diffCh chan<- diffMessage) *probe.Error {
	// Set default values for listing.
	isIncomplete := false // we will not compare any incomplete objects.
	srcCh := sourceClnt.List(globalContext, isRecursive, isIncomplete, isMetadata, dirOpt)
	tgtCh := targetClnt.List(globalContext, isRecursive, isIncomplete, isMetadata, dirOpt)

	srcCtnt, srcOk := <-srcCh
	tgtCtnt, tgtOk := <-tgtCh

	var (
		srcEOF, tgtEOF bool
	)

	for {
		srcEOF = !srcOk
		tgtEOF = !tgtOk

		// No objects from source AND target: Finish
		if srcEOF && tgtEOF {
			break
		}

		if !srcEOF && srcCtnt.Err != nil {
			return srcCtnt.Err.Trace(sourceURL, targetURL)
		}

		if !tgtEOF && tgtCtnt.Err != nil {
			return tgtCtnt.Err.Trace(sourceURL, targetURL)
		}

		// If source doesn't have objects anymore, comparison becomes obvious
		if srcEOF {
			diffCh <- diffMessage{
				SecondURL:     tgtCtnt.URL.String(),
				Diff:          differInSecond,
				secondContent: tgtCtnt,
			}
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}

		// The same for target
		if tgtEOF {
			diffCh <- diffMessage{
				FirstURL:     srcCtnt.URL.String(),
				Diff:         differInFirst,
				firstContent: srcCtnt,
			}
			srcCtnt, srcOk = <-srcCh
			continue
		}

		srcSuffix := strings.TrimPrefix(srcCtnt.URL.String(), sourceURL)
		tgtSuffix := strings.TrimPrefix(tgtCtnt.URL.String(), targetURL)

		current := urlJoinPath(targetURL, srcSuffix)
		expected := urlJoinPath(targetURL, tgtSuffix)

		if !utf8.ValidString(srcSuffix) {
			// Error. Keys must be valid UTF-8.
			diffCh <- diffMessage{Error: errInvalidSource(current).Trace()}
			srcCtnt, srcOk = <-srcCh
			continue
		}
		if !utf8.ValidString(tgtSuffix) {
			// Error. Keys must be valid UTF-8.
			diffCh <- diffMessage{Error: errInvalidTarget(expected).Trace()}
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}

		// Normalize to avoid situations where multiple byte representations are possible.
		// e.g. 'ä' can be represented as precomposed U+00E4 (UTF-8 0xc3a4) or decomposed
		// U+0061 U+0308 (UTF-8 0x61cc88).
		normalizedCurrent := norm.NFC.String(current)
		normalizedExpected := norm.NFC.String(expected)

		if normalizedExpected > normalizedCurrent {
			diffCh <- diffMessage{
				FirstURL:     srcCtnt.URL.String(),
				Diff:         differInFirst,
				firstContent: srcCtnt,
			}
			srcCtnt, srcOk = <-srcCh
			continue
		}
		if normalizedExpected == normalizedCurrent {
			srcType, tgtType := srcCtnt.Type, tgtCtnt.Type
			srcSize, tgtSize := srcCtnt.Size, tgtCtnt.Size
			if srcType.IsRegular() && !tgtType.IsRegular() ||
				!srcType.IsRegular() && tgtType.IsRegular() {
				// Type differs. Source is never a directory.
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInType,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
				continue
			}
			if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
				// Regular files differing in size.
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInSize,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			} else if activeActiveModTimeUpdated(srcCtnt, tgtCtnt) {
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInAASourceMTime,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			} else if isMetadata &&
				!metadataEqual(srcCtnt.UserMetadata, tgtCtnt.UserMetadata) &&
				!metadataEqual(srcCtnt.Metadata, tgtCtnt.Metadata) {

				// Regular files user requesting additional metadata to same file.
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInMetadata,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			}

			// No differ
			if returnSimilar {
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInNone,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			}
			srcCtnt, srcOk = <-srcCh
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}
		// Differ in second
		diffCh <- diffMessage{
			SecondURL:     tgtCtnt.URL.String(),
			Diff:          differInSecond,
			secondContent: tgtCtnt,
		}
		tgtCtnt, tgtOk = <-tgtCh
		continue
	}

	return nil
}

// objectDifference function finds the difference between all objects
// recursively in sorted order from source and target.
func difference(sourceClnt, targetClnt Client, sourceURL, targetURL string, isMetadata bool, isRecursive, returnSimilar bool, dirOpt DirOpt) (diffCh chan diffMessage) {
	diffCh = make(chan diffMessage, 10000)

	go func() {
		defer close(diffCh)

		doneCh := make(chan struct{}, 1)
		defer close(doneCh)

		for range newRetryTimerContinous(time.Second, time.Second*30, minio.MaxJitter, doneCh) {
			err := differenceInternal(sourceClnt, targetClnt, sourceURL, targetURL,
				isMetadata, isRecursive, returnSimilar, dirOpt, diffCh)
			if err != nil {
				// handle this specifically for filesystem related errors.
				switch err.ToGoError().(type) {
				case PathNotFound, PathInsufficientPermission:
					diffCh <- diffMessage{
						Error: err,
					}
					return
				}
				errorIf(err, "Unable to list comparison retrying..")
			} else {
				// Success.
				break
			}

		}
	}()

	return diffCh
}
