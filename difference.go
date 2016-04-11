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
	"strings"

	"github.com/minio/minio/pkg/probe"
)

// differType difference in type.
type differType string

const (
	differInSize   differType = "size"           // differs in size
	differInTime              = "time"           // differs in time
	differInFirst             = "only-in-first"  // only on first source
	differInSecond            = "only-in-second" // only on second source
	differInType              = "type"           // differs in type, exfile/directory
	differInNone              = ""               // does not differ
)

type differCh struct {
	dType         differType
	sourceURL     string
	targetURL     string
	sourceContent *clientContent
	targetContent *clientContent
	err           *probe.Error
}

func differenceCh(sourceClnt Client, targetClnt Client) <-chan differCh {
	dCh := make(chan differCh)
	isIncomplete := false
	isRecursive := true
	targetCh := targetClnt.List(isRecursive, isIncomplete)
	sourceCh := sourceClnt.List(isRecursive, isIncomplete)
	targetURL := targetClnt.GetURL().String()
	sourceURL := sourceClnt.GetURL().String()
	go func() {
		defer close(dCh)
		for {
			sourceContent, sourceOK := <-sourceCh
			targetContent, targetOK := <-targetCh
			if !sourceOK && !targetOK {
				break
			}
			if sourceContent != nil && sourceContent.Err != nil {
				dCh <- differCh{
					err: sourceContent.Err.Trace(sourceURL, targetURL),
				}
				break
			}
			if targetContent != nil && targetContent.Err != nil {
				dCh <- differCh{
					err: targetContent.Err.Trace(sourceURL, targetURL),
				}
				break
			}
			var sourceSuffix, targetSuffix string
			if sourceContent != nil {
				sourceSuffix = strings.TrimPrefix(sourceContent.URL.String(), sourceURL)
			}
			if targetContent != nil {
				targetSuffix = strings.TrimPrefix(targetContent.URL.String(), targetURL)
			}
			if sourceContent != nil && targetContent == nil {
				dCh <- differCh{
					dType:         differInFirst,
					sourceURL:     sourceContent.URL.String(),
					targetURL:     urlJoinPath(targetURL, sourceSuffix),
					sourceContent: sourceContent,
					err:           nil,
				}
				continue
			}
			if targetContent != nil && sourceContent == nil {
				dCh <- differCh{
					dType:         differInSecond,
					sourceURL:     urlJoinPath(sourceURL, targetSuffix),
					targetURL:     targetContent.URL.String(),
					targetContent: targetContent,
					err:           nil,
				}
				continue
			}
			if targetSuffix < sourceSuffix {
				dCh <- differCh{
					dType:         differInSecond,
					sourceURL:     urlJoinPath(sourceURL, sourceSuffix),
					targetURL:     urlJoinPath(targetURL, targetSuffix),
					sourceContent: sourceContent,
					targetContent: targetContent,
					err:           nil,
				}
			} else if sourceSuffix < targetSuffix {
				dCh <- differCh{
					dType:         differInFirst,
					sourceURL:     urlJoinPath(sourceURL, sourceSuffix),
					targetURL:     urlJoinPath(targetURL, targetSuffix),
					sourceContent: sourceContent,
					targetContent: targetContent,
					err:           nil,
				}
			}
			if sourceSuffix == targetSuffix {
				tgtType := targetContent.Type
				srcType := sourceContent.Type
				if srcType.IsRegular() && !tgtType.IsRegular() {
					// Type differs. Source is never a directory.
					dCh <- differCh{
						dType:         differInType,
						sourceURL:     urlJoinPath(sourceURL, sourceSuffix),
						targetURL:     urlJoinPath(targetURL, targetSuffix),
						sourceContent: sourceContent,
						targetContent: targetContent,
						err:           nil,
					}
					continue
				}
				tgtSize := targetContent.Size
				srcSize := sourceContent.Size
				if (srcType.IsRegular() && tgtType.IsRegular()) && srcSize != tgtSize {
					// Same type, differs in size.
					dCh <- differCh{
						dType:         differInSize,
						sourceURL:     urlJoinPath(sourceURL, sourceSuffix),
						targetURL:     urlJoinPath(targetURL, targetSuffix),
						sourceContent: sourceContent,
						targetContent: targetContent,
						err:           nil,
					}
					continue
				}
				tgtTime := targetContent.Time
				srcTime := sourceContent.Time
				if (srcType.IsRegular() && tgtType.IsRegular()) && srcTime.After(tgtTime) {
					// Same type, differs in time.
					dCh <- differCh{
						dType:         differInTime,
						sourceURL:     urlJoinPath(sourceURL, sourceSuffix),
						targetURL:     urlJoinPath(targetURL, targetSuffix),
						sourceContent: sourceContent,
						targetContent: targetContent,
						err:           nil,
					}
				}
			}
		}
	}()
	return dCh
}
