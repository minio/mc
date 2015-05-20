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

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
)

type diff struct {
	message string
	err     error
}

// urlJoinPath Join a path to existing URL
func urlJoinPath(urlStr, path string) (newURLStr string, err error) {
	u, err := client.Parse(urlStr)
	if err != nil {
		return "", iodine.New(err, nil)
	}

	u.Path = filepath.Join(u.Path, path)
	newURLStr = u.String()
	return newURLStr, nil
}

// doDiffObjects - Diff two object URLs
func doDiffObjects(firstURL, secondURL string, ch chan diff) {
	if firstURL == secondURL {
		return
	}
	_, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + firstURL + "’ " + "Reason: [" + iodine.ToError(err).Error() + "].\n",
			err:     iodine.New(err, nil),
		}
		return
	}

	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + secondURL + "’ " + "Reason: [" + iodine.ToError(err).Error() + "].\n",
			err:     iodine.New(err, nil),
		}
		return
	}

	switch {
	case firstContent.Type.IsRegular():
		if !secondContent.Type.IsRegular() {
			ch <- diff{
				message: firstURL + " and " + secondURL + " differs in type.\n",
				err:     nil,
			}
		}
	default:
		ch <- diff{
			message: "‘" + firstURL + "’ is not an object. Please report this bug with ‘--debug’ option\n.",
			err:     iodine.New(errNotAnObject{url: firstURL}, nil),
		}
	}

	if firstContent.Size != secondContent.Size {
		ch <- diff{
			message: firstURL + " and " + secondURL + " differs in size.\n",
			err:     nil,
		}
	}
}

// doDiffDirs - Diff two Dir URLs
func doDiffDirs(firstURL, secondURL string, ch chan diff) {
	firstClnt, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + firstURL + "’ ." + "Reason: [" + iodine.ToError(err).Error() + "].\n",
			err:     iodine.New(err, nil),
		}
		return
	}
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + secondURL + "’ ." + "Reason: [" + iodine.ToError(err).Error() + "].\n",
			err:     iodine.New(err, nil),
		}
		return
	}
	switch {
	case firstContent.Type.IsDir():
		if !secondContent.Type.IsDir() {
			ch <- diff{
				message: firstURL + " and " + secondURL + " differs in type.\n",
				err:     nil,
			}
		}
	default:
		ch <- diff{
			message: "‘" + firstURL + "’ is not an object. Please report this bug with ‘--debug’ option\n.",
			err:     iodine.New(errNotAnObject{url: firstURL}, nil),
		}
	}
	for contentCh := range firstClnt.List() {
		if contentCh.Err != nil {
			ch <- diff{
				message: "Failed to list ‘" + firstURL + "’. Reason: [" + iodine.ToError(contentCh.Err).Error() + "].\n",
				err:     iodine.New(contentCh.Err, nil),
			}
			return
		}
		newFirstURL, err := urlJoinPath(firstURL, contentCh.Content.Name)
		if err != nil {
			ch <- diff{
				message: "Unable to construct new URL from ‘" + firstURL + "’ using ‘" + contentCh.Content.Name + "’. Reason: [" + iodine.ToError(err).Error() + "].\n",
				err:     iodine.New(err, nil),
			}
			return
		}
		newSecondURL, err := urlJoinPath(secondURL, contentCh.Content.Name)
		if err != nil {
			ch <- diff{
				message: "Unable to construct new URL from ‘" + secondURL + "’ using ‘" + contentCh.Content.Name + "’. Reason: [" + iodine.ToError(err).Error() + "].\n",
				err:     iodine.New(err, nil),
			}
			return
		}
		_, newFirstContent, err := url2Stat(newFirstURL)
		if err != nil {
			ch <- diff{
				message: "Failed to stat ‘" + newFirstURL + "’. Reason: [" + iodine.ToError(err).Error() + "].\n",
				err:     iodine.New(err, nil),
			}
			return

		}
		_, newSecondContent, err := url2Stat(newSecondURL)
		if err != nil {
			ch <- diff{
				message: "‘" + filepath.Base(newFirstContent.Name) + "’ only in ‘" + firstURL + "’.\n",
				err:     nil,
			}
			continue
		}
		switch {
		case newFirstContent.Type.IsDir():
			if !newSecondContent.Type.IsDir() {
				ch <- diff{
					message: newFirstURL + " and " + newSecondURL + " differs in type.\n",
					err:     nil,
				}
				continue
			}
		case newFirstContent.Type.IsRegular():
			if !newSecondContent.Type.IsRegular() {
				ch <- diff{
					message: newFirstURL + " and " + newSecondURL + " differs in type.\n",
					err:     nil,
				}
				continue
			}
			doDiffObjects(newFirstURL, newSecondURL, ch)
		}
	} // End of for-loop
}
