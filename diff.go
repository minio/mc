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
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

//
//   NOTE: All the parse rules should reduced to 1: Diff(First, Second).
//
//   Valid cases
//   =======================
//   1: diff(f, f) -> diff(f, f)
//   2: diff(f, d) -> copy(f, d/f) -> 1
//   3: diff(d1..., d2) -> []diff(d1/f, d2/f) -> []1
//
//   InValid cases
//   =======================
//   1. diff(d1..., d2) -> INVALID
//   2. diff(d1..., d2...) -> INVALID
//

type diff struct {
	message string
	err     error
}

// urlJoinPath Join a path to existing URL
func urlJoinPath(url1, url2 string) (newURLStr string, err error) {
	u1, err := client.Parse(url1)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	u2, err := client.Parse(url2)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	u1.Path = filepath.Join(u1.Path, u2.Path)
	newURLStr = u1.String()
	return newURLStr, nil
}

// doDiffObjects - Diff two object URLs
func doDiffObjects(firstURL, secondURL string, ch chan diff) {
	_, firstContent, errFirst := url2Stat(firstURL)
	_, secondContent, errSecond := url2Stat(secondURL)

	switch {
	case errFirst != nil && errSecond == nil:
		ch <- diff{
			message: "Only in ‘" + secondURL + "’",
			err:     nil,
		}
		return
	case errFirst == nil && errSecond != nil:
		ch <- diff{
			message: "Only in ‘" + firstURL + "’",
			err:     nil,
		}
		return
	}
	if firstContent.Name == secondContent.Name {
		return
	}
	switch {
	case firstContent.Type.IsRegular():
		if !secondContent.Type.IsRegular() {
			ch <- diff{
				message: firstURL + " and " + secondURL + " differs in type.",
				err:     nil,
			}
		}
	default:
		ch <- diff{
			message: "‘" + firstURL + "’ is not an object. Please report this bug with ‘--debug’ option.",
			err:     iodine.New(errNotAnObject{url: firstURL}, nil),
		}
		return
	}

	if firstContent.Size != secondContent.Size {
		ch <- diff{
			message: firstURL + " and " + secondURL + " differs in size.",
			err:     nil,
		}
	}
}

func dodiffdirs(firstClnt client.Client, firstURL, secondURL string, recursive bool, ch chan diff) {
	for contentCh := range firstClnt.List(recursive) {
		if contentCh.Err != nil {
			ch <- diff{
				message: "Failed to list ‘" + firstURL + "’",
				err:     iodine.New(contentCh.Err, nil),
			}
			return
		}
		if recursive {
			// this special handling is necessary since we are sending back absolute paths with in ListRecursive()
			//
			// To be consistent we have to filter them out
			contentCh.Content.Name = strings.TrimPrefix(contentCh.Content.Name, strings.TrimSuffix(firstURL, "/")+"/")
		}
		newFirstURL, err := urlJoinPath(firstURL, contentCh.Content.Name)
		if err != nil {
			ch <- diff{
				message: "Unable to construct new URL from ‘" + firstURL + "’ using ‘" + contentCh.Content.Name + "’",
				err:     iodine.New(err, nil),
			}
			return
		}
		newSecondURL, err := urlJoinPath(secondURL, contentCh.Content.Name)
		if err != nil {
			ch <- diff{
				message: "Unable to construct new URL from ‘" + secondURL + "’ using ‘" + contentCh.Content.Name + "’",
				err:     iodine.New(err, nil),
			}
			return
		}
		_, newFirstContent, errFirst := url2Stat(newFirstURL)
		_, newSecondContent, errSecond := url2Stat(newSecondURL)
		switch {
		case errFirst != nil && errSecond == nil:
			ch <- diff{
				message: "‘" + newSecondURL + "’ Only in ‘" + secondURL + "’",
				err:     nil,
			}
			continue
		case errFirst == nil && errSecond != nil:
			ch <- diff{
				message: "‘" + newFirstURL + "’ Only in ‘" + firstURL + "’",
				err:     nil,
			}
			continue
		case errFirst == nil && errSecond == nil:
			switch {
			case newFirstContent.Type.IsDir():
				if !newSecondContent.Type.IsDir() {
					ch <- diff{
						message: newFirstURL + " and " + newSecondURL + " differs in type.",
						err:     nil,
					}
				}
				continue
			case newFirstContent.Type.IsRegular():
				if !newSecondContent.Type.IsRegular() {
					ch <- diff{
						message: newFirstURL + " and " + newSecondURL + " differs in type.",
						err:     nil,
					}
					continue
				}
				doDiffObjects(newFirstURL, newSecondURL, ch)
			}
		}
	} // End of for-loop
}

// doDiffDirs - Diff two Dir URLs
func doDiffDirs(firstURL, secondURL string, recursive bool, ch chan diff) {
	firstClnt, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + firstURL + "’",
			err:     iodine.New(err, nil),
		}
		return
	}
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + secondURL + "’",
			err:     iodine.New(err, nil),
		}
		return
	}
	switch {
	case firstContent.Type.IsDir():
		if !secondContent.Type.IsDir() {
			ch <- diff{
				message: firstURL + " and " + secondURL + " differs in type.",
				err:     nil,
			}
		}
	default:
		ch <- diff{
			message: "‘" + firstURL + "’ is not an object. Please report this bug with ‘--debug’ option.",
			err:     iodine.New(errNotAnObject{url: firstURL}, nil),
		}
		return
	}
	dodiffdirs(firstClnt, firstURL, secondURL, recursive, ch)
}
