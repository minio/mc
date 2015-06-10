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
	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

//
//   NOTE: All the parse rules should reduced to A: Copy(Source, Target).
//
//   * SINGLE SOURCE - VALID
//   =======================
//   A: sync(f, f) -> copy(f, f)
//   B: sync(f, d) -> copy(f, d/f) -> A
//   C: sync(f, []d) -> []copy(f, d/f) -> []A
//   D: sync(d1..., d2) -> []copy(d1/f, d2/d1/f) -> []A
//   E: sync(d1..., []d2) -> [][]copy(d1/f, d2/d1/f) -> [][]A
//
//   * SINGLE SOURCE - INVALID
//   =========================
//   sync(d, *)
//   sync(d..., f)
//   sync(*, d...)
//
//   * MULTI-TARGET RECURSIVE - INVALID
//   ==================================
//   sync(*, f1)
//   sync(*, []f1)

type syncURLs struct {
	SourceContent *client.Content
	TargetContent *client.Content
	Error         error
}

// prepareCopyURLs - prepares target and source URLs for syncing.
func prepareSyncURLs(sourceURL string, targetURLs []string) <-chan syncURLs {
	syncURLsCh := make(chan syncURLs)
	go func() {
		defer close(syncURLsCh)
		switch guessCopyURLType([]string{sourceURL}, targetURLs[0]) {
		case cpURLsTypeA:
			var sURLs syncURLs
			for _, targetURL := range targetURLs {
				cpURLs := prepareCopyURLsTypeA(sourceURL, targetURL)
				sURLs.SourceContent = cpURLs.SourceContent
				sURLs.TargetContent = sURLs.TargetContent
				syncURLsCh <- sURLs
			}
		case cpURLsTypeB:
			var sURLs syncURLs
			for _, targetURL := range targetURLs {
				cpURLs := prepareCopyURLsTypeB(sourceURL, targetURL)
				sURLs.SourceContent = cpURLs.SourceContent
				sURLs.TargetContent = cpURLs.TargetContent
				syncURLsCh <- sURLs
			}
		case cpURLsTypeC:
			// collection of channels
			var cpURLsChs []<-chan *cpURLs
			for _, targetURL := range targetURLs {
				cpURLsChs = append(cpURLsChs, prepareCopyURLsTypeC(sourceURL, targetURL))
			}
			for _, cpURLsCh := range cpURLsChs {
				var sURLs syncURLs
				for cpURLs := range cpURLsCh {
					if cpURLs.Error != nil {
						syncURLsCh <- syncURLs{Error: iodine.New(cpURLs.Error, nil)}
						break
					}
					sURLs.SourceContent = cpURLs.SourceContent
					sURLs.TargetContent = cpURLs.TargetContent
					syncURLsCh <- sURLs
				}
			}
		default:
			syncURLsCh <- syncURLs{Error: iodine.New(errInvalidArgument{}, nil)}
		}
	}()
	return syncURLsCh
}
