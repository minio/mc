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
	differInSize  differType = "size"          // differs in size
	differInTime             = "time"          // differs in time
	differInFirst            = "only-in-first" // only on first source
	differInType             = "type"          // differs in type, exfile/directory
	differInNone             = ""              // does not differ
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
				tgtTime := content.Time
				if srcType.IsRegular() && !tgtType.IsRegular() {
					// Type differes. Source is never a directory.
					return differInType, nil
				}
				if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
					// Regular files differing in size.
					return differInSize, nil
				}
				if (srcType.IsRegular() && tgtType.IsRegular()) && srcTime.After(tgtTime) {
					// Regular files differing in time.
					return differInTime, nil
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
