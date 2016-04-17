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
	"fmt"
	"strings"
)

// differType difference in type.
type differType int

const (
	differInNone   differType = iota // does not differ
	differInSize                     // differs in size
	differInType                     // only in source
	differInFirst                    // only in target
	differInSecond                   // differs in type, exfile/directory
)

func (d differType) String() string {
	switch d {
	case differInNone:
		return ""
	case differInSize:
		return "size"
	case differInType:
		return "type"
	case differInFirst:
		return "only-in-first"
	case differInSecond:
		return "only-in-second"
	}
	return "unknown"
}

// objectDifference function finds the difference between all objects
// recursively in sorted order from source and target.
func objectDifference(sourceClnt, targetClnt Client, sourceURL, targetURL string) (diffCh chan diffMessage) {
	var (
		srcEOF, tgtEOF       bool
		srcOk, tgtOk         bool
		srcCtnt, tgtCtnt     *clientContent
		srcSuffix, tgtSuffix string
	)

	// Set default values for listing.
	isRecursive := true   // recursive is always true for diff.
	isIncomplete := false // we will not compare any incomplete objects.
	srcCh := sourceClnt.List(isRecursive, isIncomplete)
	tgtCh := targetClnt.List(isRecursive, isIncomplete)

	diffCh = make(chan diffMessage, 1000)

	go func() {

		srcCtnt, srcOk = <-srcCh
		tgtCtnt, tgtOk = <-tgtCh

		for {
			srcEOF = !srcOk
			tgtEOF = !tgtOk

			// No objects from source AND target: Finish
			if srcEOF && tgtEOF {
				close(diffCh)
				break
			}

			if !srcEOF && srcCtnt.Err != nil {
				switch srcCtnt.Err.ToGoError().(type) {
				// Handle this specifically for filesystem related errors.
				case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
					errorIf(srcCtnt.Err.Trace(sourceURL, targetURL), fmt.Sprintf("Failed on '%s'", sourceURL))
				default:
					fatalIf(srcCtnt.Err.Trace(sourceURL, targetURL), fmt.Sprintf("Failed on '%s'", sourceURL))
				}
				srcCtnt, srcOk = <-srcCh
				continue
			}

			if !tgtEOF && tgtCtnt.Err != nil {
				switch tgtCtnt.Err.ToGoError().(type) {
				// Handle this specifically for filesystem related errors.
				case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
					errorIf(tgtCtnt.Err.Trace(sourceURL, targetURL), fmt.Sprintf("Failed on '%s'", targetURL))
				default:
					fatalIf(tgtCtnt.Err.Trace(sourceURL, targetURL), fmt.Sprintf("Failed on '%s'", targetURL))
				}
				tgtCtnt, tgtOk = <-tgtCh
				continue
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

			srcSuffix = strings.TrimPrefix(srcCtnt.URL.String(), sourceURL)
			tgtSuffix = strings.TrimPrefix(tgtCtnt.URL.String(), targetURL)

			current := urlJoinPath(targetURL, srcSuffix)
			expected := urlJoinPath(targetURL, tgtSuffix)

			if expected > current {
				diffCh <- diffMessage{
					FirstURL:     srcCtnt.URL.String(),
					Diff:         differInFirst,
					firstContent: srcCtnt,
				}
				srcCtnt, srcOk = <-srcCh
				continue
			}
			if expected == current {
				srcType, tgtType := srcCtnt.Type, tgtCtnt.Type
				srcSize, tgtSize := srcCtnt.Size, tgtCtnt.Size
				if srcType.IsRegular() && !tgtType.IsRegular() ||
					!srcType.IsRegular() && tgtType.IsRegular() {
					// Type differes. Source is never a directory.
					diffCh <- diffMessage{
						FirstURL:      srcCtnt.URL.String(),
						SecondURL:     tgtCtnt.URL.String(),
						Diff:          differInType,
						firstContent:  srcCtnt,
						secondContent: tgtCtnt,
					}
				} else if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
					// Regular files differing in size.
					diffCh <- diffMessage{
						FirstURL:      srcCtnt.URL.String(),
						SecondURL:     tgtCtnt.URL.String(),
						Diff:          differInSize,
						firstContent:  srcCtnt,
						secondContent: tgtCtnt,
					}
				}
				// No differ
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
	}()

	return diffCh
}
