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
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

//
//   NOTE: All the parse rules should reduced to 1: Diff(First, Second).
//
//   Valid cases
//   =======================
//   1: diff(f, f) -> diff(f, f)
//   2: diff(f, d) -> diff(f, d/f) -> 1
//   3. diff(d1, d2) -> []diff(d1/f, d2/f) -> []1
//   4: diff(d1..., d2) -> []diff(d1/f, d2/f) -> []1
//
//   Invalid cases
//   =======================
//   1. diff(d1..., d2/f) -> INVALID
//   2. diff(d1..., d2...) -> INVALID
//

// diffMessage json container for diff messages
type diffMessage struct {
	FirstURL  string       `json:"first"`
	SecondURL string       `json:"second"`
	Diff      string       `json:"diff"`
	Error     *probe.Error `json:"error,omitempty"`
}

// String colorized diff message
func (d diffMessage) String() string {
	msg := ""
	switch d.Diff {
	case "only-in-first":
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffOnlyInFirst", " - only in first.")
	case "type":
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffType", " - differ in type.")
	case "size":
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffSize", " - differ in size.")
	default:
		fatalIf(errDummy().Trace(),
			"Unhandled difference between ‘"+d.FirstURL+"’ and ‘"+d.SecondURL+"’.")
	}
	return msg

}

// JSON jsonified diff message
func (d diffMessage) JSON() string {
	diffJSONBytes, err := json.Marshal(d)
	fatalIf(probe.NewError(err),
		"Unable to marshal diff message ‘"+d.FirstURL+"’, ‘"+d.SecondURL+"’ and ‘"+d.Diff+"’.")
	return string(diffJSONBytes)
}

// diffObjects - diff two incoming object contents, this is the most basic types.
//
// 1: diff(f, f) -> diff(f, f) -> VALID
// 2: diff(f, d) -> diff(f, d/f) -> VALID
func diffObjects(firstContent, secondContent *client.Content) *diffMessage {
	if firstContent.URL.String() == secondContent.URL.String() {
		return nil
	}
	if firstContent.Type.IsRegular() && secondContent.Type.IsRegular() {
		if firstContent.Size != secondContent.Size {
			return &diffMessage{
				FirstURL:  firstContent.URL.String(),
				SecondURL: secondContent.URL.String(),
				Diff:      "size",
			}
		}
		return nil
	}
	if firstContent.Type.IsRegular() && secondContent.Type.IsDir() {
		newSecondURLStr := urlJoinPath(secondContent.URL.String(), firstContent.URL.String())
		_, newSecondContent, err := url2Stat(newSecondURLStr)
		if err != nil {
			return &diffMessage{
				FirstURL:  firstContent.URL.String(),
				SecondURL: newSecondURLStr,
				Diff:      "only-in-first",
			}
		}
		if firstContent.Size != newSecondContent.Size {
			return &diffMessage{
				FirstURL:  firstContent.URL.String(),
				SecondURL: newSecondContent.URL.String(),
				Diff:      "size",
			}
		}
		return nil
	}
	if firstContent.Type.IsRegular() && !secondContent.Type.IsRegular() {
		return &diffMessage{
			FirstURL:  firstContent.URL.String(),
			SecondURL: secondContent.URL.String(),
			Diff:      "type",
		}
	}
	return nil
}

// diffFolders - diff of contents of two folders only top level content.
//
// 3: diff(d1, d2) -> []diff(d1/f, d2/f) -> VALID
func diffFolders(firstClnt, secondClnt client.Client, outCh chan<- diffMessage) {
	recursive := false
	// Range on the List to consume incoming content
	for contentCh := range firstClnt.List(recursive, false) {
		if contentCh.Err != nil {
			outCh <- diffMessage{
				Error: contentCh.Err.Trace(firstClnt.GetURL().String()),
			}
			continue
		}
		// Store incoming content
		newFirstContent := contentCh.Content
		newFirstURLStr := newFirstContent.URL.String()
		// Construct the second URL.
		newSecondURL := secondClnt.GetURL()
		// Need to verify the same path from first URL, construct the second URL
		newSecondURL.Path = filepath.Join(newSecondURL.Path, filepath.Base(contentCh.Content.URL.Path))
		newSecondURLStr := newSecondURL.String()
		// Send a stat to verify
		_, newSecondContent, err := url2Stat(newSecondURLStr)
		if err != nil {
			outCh <- diffMessage{
				FirstURL:  newFirstURLStr,
				SecondURL: newSecondURLStr,
				Diff:      "only-in-first",
			}
			continue
		}
		diffMsg := diffObjects(newFirstContent, newSecondContent)
		if diffMsg != nil {
			outCh <- *diffMsg
			continue
		}
	} // Reached EOF
}

// diffFoldersRecursive diff folders for all files recursively.
//
// 4: diff(d1..., d2) -> []diff(d1/f, d2/f) -> VALID.
func diffFoldersRecursive(firstClnt, secondClnt client.Client, outCh chan<- diffMessage) {
	var scanBar scanBarFunc
	if !globalQuietFlag && !globalJSONFlag { // set up progress bar
		scanBar = scanBarFactory()
	}
	recursive := true
	firstListCh := firstClnt.List(recursive, false) // Copy first list channel.
	for firstContentCh := range firstListCh {
		if firstContentCh.Err != nil {
			outCh <- diffMessage{Error: firstContentCh.Err.Trace()}
			continue
		}
		if firstContentCh.Content.Type.IsDir() {
			// Skip directories there is no concept of directories on S3.
			continue
		}
		firstContent := firstContentCh.Content
		secondURL := secondClnt.GetURL()
		secondURL.Path = filepath.Join(secondURL.Path,
			strings.TrimPrefix(firstContent.URL.Path, url2Dir(firstClnt.GetURL().Path)))
		_, secondContent, err := url2Stat(secondURL.String())
		if err != nil {
			outCh <- diffMessage{
				FirstURL:  firstContent.URL.String(),
				SecondURL: secondURL.String(),
				Diff:      "only-in-first",
			}
			continue
		}
		if diffMsg := diffObjects(firstContent, secondContent); diffMsg != nil {
			outCh <- *diffMsg
			continue
		}
		if !globalQuietFlag && !globalJSONFlag { // set up progress bar
			scanBar(firstContent.URL.String())
		}
	}
}
