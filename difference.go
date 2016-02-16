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

	"github.com/minio/minio/pkg/probe"
)

// objectDifference function finds the difference between object on source and target
// it takes suffix string, type and size on the source
// objectDifferenceFactory returns objectDifference function
type objectDifference func(string, os.FileMode, int64) (string, *probe.Error)

const (
	differSize      string = "size"          // differs in size
	differOnlyFirst string = "only-in-first" // only on source
	differType      string = "type"          // differs in type, ex file/directory
	differNone      string = ""              // does not differ
)

// objectDifferenceFactory returns objectDifference function to check for difference
// between sourceURL and targetURL
// for usage reference check diff and mirror commands
func objectDifferenceFactory(targetAlias, targetURL string) (objectDifference, *probe.Error) {
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		return nil, err.Trace(targetAlias, targetURL)
	}
	isIncomplete := false
	isRecursive := true
	ch := clnt.List(isRecursive, isIncomplete)
	current := targetURL
	reachedEOF := false
	ok := false
	var content *clientContent

	difference := func(suffix string, srcType os.FileMode, srcSize int64) (string, *probe.Error) {
		if reachedEOF {
			// would mean the suffix is not on target
			return differOnlyFirst, nil
		}
		expected := urlJoinPath(targetURL, suffix)
		for {
			if expected < current {
				return differOnlyFirst, nil // not available in the target
			}
			if expected == current {
				tgtType := content.Type
				tgtSize := content.Size
				if srcType.IsRegular() && !tgtType.IsRegular() {
					// Type differes. Source is never a directory
					return differType, nil
				}
				if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
					// regular files differing in size
					return differSize, nil
				}
				return differNone, nil // available in the target
			}
			content, ok = <-ch
			if !ok {
				reachedEOF = true
				return differOnlyFirst, nil
			}
			if content.Err != nil {
				return "", content.Err.Trace()
			}
			current = content.URL.String()
		}
	}
	return difference, nil
}
