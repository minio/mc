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
	"strings"
	"unicode/utf8"

	// golang does not support flat keys for path matching, find does

	"golang.org/x/text/unicode/norm"
)

// differType difference in type.
type differType int

const (
	differInNone   differType = iota // does not differ
	differInSize                     // differs in size
	differInTime                     // differs in time
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
	case differInTime:
		return "time"
	case differInType:
		return "type"
	case differInFirst:
		return "only-in-first"
	case differInSecond:
		return "only-in-second"
	}
	return "unknown"
}

func objectDifference(sourceClnt, targetClnt Client, sourceURL, targetURL string) (diffCh chan diffMessage) {
	return difference(sourceClnt, targetClnt, sourceURL, targetURL, true, false, DirNone)
}

func dirDifference(sourceClnt, targetClnt Client, sourceURL, targetURL string) (diffCh chan diffMessage) {
	return difference(sourceClnt, targetClnt, sourceURL, targetURL, false, true, DirFirst)
}

// objectDifference function finds the difference between all objects
// recursively in sorted order from source and target.
func difference(sourceClnt, targetClnt Client, sourceURL, targetURL string, isRecursive, returnSimilar bool, dirOpt DirOpt) (diffCh chan diffMessage) {
	var (
		srcEOF, tgtEOF       bool
		srcOk, tgtOk         bool
		srcCtnt, tgtCtnt     *clientContent
		srcSuffix, tgtSuffix string
	)

	// Set default values for listing.
	isIncomplete := false // we will not compare any incomplete objects.
	srcCh := sourceClnt.List(isRecursive, isIncomplete, dirOpt)
	tgtCh := targetClnt.List(isRecursive, isIncomplete, dirOpt)

	diffCh = make(chan diffMessage, 1000)

	go func() {

		srcCtnt, srcOk = <-srcCh
		tgtCtnt, tgtOk = <-tgtCh

		defer close(diffCh)

		for {
			srcEOF = !srcOk
			tgtEOF = !tgtOk

			// No objects from source AND target: Finish
			if srcEOF && tgtEOF {
				break
			}

			if !srcEOF && srcCtnt.Err != nil {
				diffCh <- diffMessage{Error: srcCtnt.Err.Trace(sourceURL, targetURL)}
				return
			}

			if !tgtEOF && tgtCtnt.Err != nil {
				diffCh <- diffMessage{Error: tgtCtnt.Err.Trace(sourceURL, targetURL)}
				return
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
				srcTime, tgtTime := srcCtnt.Time, tgtCtnt.Time
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
				} else if srcTime.After(tgtTime) {
					// Regular files differing in timestamp.
					diffCh <- diffMessage{
						FirstURL:      srcCtnt.URL.String(),
						SecondURL:     tgtCtnt.URL.String(),
						Diff:          differInTime,
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
	}()

	return diffCh
}
