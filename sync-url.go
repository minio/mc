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

type syncURLs copyURLs

// prepareCopyURLs - prepares target and source URLs for syncing.
func prepareSyncURLs(sourceURL string, targetURLs []string) <-chan *copyURLs {
	syncURLsCh := make(chan *copyURLs)

	go func() {
		defer close(syncURLsCh)
		for _, targetURL := range targetURLs {
			switch guessCopyURLType([]string{sourceURL}, targetURL) {
			case copyURLsTypeA:
				syncURLs := prepareCopyURLsTypeA(sourceURL, targetURL)
				syncURLsCh <- syncURLs
			case copyURLsTypeB:
				syncURLs := prepareCopyURLsTypeB(sourceURL, targetURL)
				syncURLsCh <- syncURLs
			case copyURLsTypeC:
				for syncURLs := range prepareCopyURLsTypeC(sourceURL, targetURL) {
					syncURLsCh <- syncURLs
				}
			default:
				syncURLsCh <- &copyURLs{Error: iodine.New(errors.New("Invalid arguments."), nil)}
			}
		}
	}()
	return syncURLsCh
}
