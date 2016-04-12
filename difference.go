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
	"os"
	"strings"
	"time"

	"github.com/minio/minio/pkg/probe"
)

// objectDifference function finds the difference between object on
// source and target it takes suffix string, type and size on the
// source objectDifferenceFactory returns objectDifference function
type objectDifference func(string, string, os.FileMode, int64, time.Time) (differType, *probe.Error)

// differType difference in type.
type differType string

const (
	differInSize   differType = "size"           // differs in size
	differInFirst             = "only-in-first"  // only in source
	differInSecond            = "only-in-second" // only in target
	differInType              = "type"           // differs in type, exfile/directory
	differInNone              = ""               // does not differ
)

// objectDifferenceFactory returns objectDifference function to check for difference
// between sourceURL and targetURL, for usage reference check diff and mirror commands.
func objectDifferenceFactory(targetClnt Client) objectDifference {
	isIncomplete := false
	isRecursive := true
	ch := targetClnt.List(isRecursive, isIncomplete)
	reachedEOF := false
	ok := false
	var content *clientContent

	return func(targetURL string, srcSuffix string, srcType os.FileMode, srcSize int64, srcTime time.Time) (differType, *probe.Error) {
		if reachedEOF {
			// Would mean the suffix is not on target.
			return differInFirst, nil
		}
		current := targetURL
		expected := urlJoinPath(targetURL, srcSuffix)
		for {
			if expected < current {
				return differInFirst, nil // Not available in the target.
			}
			if expected == current {
				tgtType := content.Type
				tgtSize := content.Size
				if srcType.IsRegular() && !tgtType.IsRegular() {
					// Type differes. Source is never a directory.
					return differInType, nil
				}
				if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
					// Regular files differing in size.
					return differInSize, nil
				}
				return differInNone, nil // Available in the target.
			}
			content, ok = <-ch
			if !ok {
				reachedEOF = true
				return differInFirst, nil
			}
			if content.Err != nil {
				return "", content.Err.Trace()
			}
			current = content.URL.String()
		}
	}
}

func objectDifferenceNewImpl(sourceClnt, targetClnt Client, sourceURL, targetURL string) (d chan diffMessage) {

	var (
		srcEOF, tgtEOF       bool
		srcOk, tgtOk         bool
		srcCtnt, tgtCtnt     *clientContent
		srcSuffix, tgtSuffix string
	)

	isIncomplete := false
	isRecursive := true
	srcCh := sourceClnt.List(isRecursive, isIncomplete)
	tgtCh := targetClnt.List(isRecursive, isIncomplete)

	d = make(chan diffMessage, 1000)

	go func() {

		srcCtnt, srcOk = <-srcCh
		tgtCtnt, tgtOk = <-tgtCh

		for {
			srcEOF = !srcOk
			tgtEOF = !tgtOk

			// No objects from source AND target: Finish
			if srcEOF && tgtEOF {
				close(d)
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
				d <- diffMessage{
					SecondURL: tgtCtnt.URL.String(),
					Diff:      differInSecond,
				}
				tgtCtnt, tgtOk = <-tgtCh
				continue
			}

			// The same for target
			if tgtEOF {
				d <- diffMessage{
					FirstURL: srcCtnt.URL.String(),
					Diff:     differInFirst,
				}
				srcCtnt, srcOk = <-srcCh
				continue
			}

			srcSuffix = strings.TrimPrefix(srcCtnt.URL.String(), sourceURL)
			tgtSuffix = strings.TrimPrefix(tgtCtnt.URL.String(), targetURL)

			current := urlJoinPath(targetURL, srcSuffix)
			expected := urlJoinPath(targetURL, tgtSuffix)

			if expected > current {
				d <- diffMessage{
					FirstURL: srcCtnt.URL.String(),
					Diff:     differInFirst,
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
					d <- diffMessage{
						FirstURL:  srcCtnt.URL.String(),
						SecondURL: tgtCtnt.URL.String(),
						Diff:      differInType,
					}
				} else if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
					// Regular files differing in size.
					d <- diffMessage{
						FirstURL:  srcCtnt.URL.String(),
						SecondURL: tgtCtnt.URL.String(),
						Diff:      differInSize,
					}
				}
				// No differ
				srcCtnt, srcOk = <-srcCh
				tgtCtnt, tgtOk = <-tgtCh
				continue
			}
			// Differ in second
			d <- diffMessage{
				SecondURL: tgtCtnt.URL.String(),
				Diff:      differInSecond,
			}
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}
	}()

	return d
}
