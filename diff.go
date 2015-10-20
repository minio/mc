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
//   2: diff(f, d) -> copy(f, d/f) -> 1
//   3: diff(d1..., d2) -> []diff(d1/f, d2/f) -> []1
//
//   InValid cases
//   =======================
//   1. diff(d1..., d2) -> INVALID
//   2. diff(d1..., d2...) -> INVALID
//

// DiffMessage json container for diff messages
type DiffMessage struct {
	FirstURL  string       `json:"first"`
	SecondURL string       `json:"second"`
	Diff      string       `json:"diff"`
	Error     *probe.Error `json:"error,omitempty"`
}

// String colorized diff message
func (d DiffMessage) String() string {
	msg := ""
	switch d.Diff {
	case "only-in-first":
		msg = console.Colorize("DiffMessage", "‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffOnlyInFirst", " - only in first.")
	case "type":
		msg = console.Colorize("DiffMessage", "‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffType", " - differ in type.")
	case "size":
		msg = console.Colorize("DiffMessage", "‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffSize", " - differ in size.")
	default:
		fatalIf(errDummy().Trace(), "Unhandled difference between ‘"+d.FirstURL+"’ and ‘"+d.SecondURL+"’.")
	}
	return msg

}

// JSON jsonified diff message
func (d DiffMessage) JSON() string {
	diffJSONBytes, err := json.Marshal(d)
	fatalIf(probe.NewError(err), "Unable to marshal diff message ‘"+d.FirstURL+"’, ‘"+d.SecondURL+"’ and ‘"+d.Diff+"’.")

	return string(diffJSONBytes)
}

func doDiffInRoutine(firstURL, secondURL string, recursive bool, ch chan DiffMessage) {
	defer close(ch)
	firstClnt, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- DiffMessage{
			Error: err.Trace(firstURL),
		}
		return
	}
	secondClnt, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- DiffMessage{
			Error: err.Trace(secondURL),
		}
		return
	}
	if firstContent.Type.IsRegular() {
		switch {
		case secondContent.Type.IsDir():
			newSecondURL := urlJoinPath(secondURL, firstURL)
			doDiffObjects(firstURL, newSecondURL, ch)
		case !secondContent.Type.IsRegular():
			ch <- DiffMessage{
				FirstURL:  firstURL,
				SecondURL: secondURL,
				Diff:      "type",
			}
			return
		case secondContent.Type.IsRegular():
			doDiffObjects(firstURL, secondURL, ch)
		}
	}
	if firstContent.Type.IsDir() {
		switch {
		case !secondContent.Type.IsDir():
			ch <- DiffMessage{
				FirstURL:  firstURL,
				SecondURL: secondURL,
				Diff:      "type",
			}
			return
		default:
			doDiffDirs(firstClnt, secondClnt, recursive, ch)
		}
	}
}

// doDiffObjects - Diff two object URLs
func doDiffObjects(firstURL, secondURL string, ch chan DiffMessage) {
	_, firstContent, errFirst := url2Stat(firstURL)
	_, secondContent, errSecond := url2Stat(secondURL)

	switch {
	case errFirst != nil && errSecond == nil:
		ch <- DiffMessage{
			Error: errFirst.Trace(firstURL, secondURL),
		}
		return
	case errFirst == nil && errSecond != nil:
		ch <- DiffMessage{
			Error: errSecond.Trace(firstURL, secondURL),
		}
		return
	}
	if firstContent.Name == secondContent.Name {
		return
	}
	switch {
	case firstContent.Type.IsRegular():
		if !secondContent.Type.IsRegular() {
			ch <- DiffMessage{
				FirstURL:  firstURL,
				SecondURL: secondURL,
				Diff:      "type",
			}
		}
	default:
		ch <- DiffMessage{
			Error: errNotAnObject(firstURL).Trace(),
		}
		return
	}

	if firstContent.Size != secondContent.Size {
		ch <- DiffMessage{
			FirstURL:  firstURL,
			SecondURL: secondURL,
			Diff:      "size",
		}
	}
}

func dodiff(firstClnt, secondClnt client.Client, ch chan DiffMessage) {
	for contentCh := range firstClnt.List(false, false) {
		if contentCh.Err != nil {
			ch <- DiffMessage{
				Error: contentCh.Err.Trace(firstClnt.URL().String()),
			}
			return
		}
		newFirstURL := urlJoinPath(firstClnt.URL().String(), contentCh.Content.Name)
		newSecondURL := urlJoinPath(secondClnt.URL().String(), contentCh.Content.Name)
		_, newFirstContent, errFirst := url2Stat(newFirstURL)
		_, newSecondContent, errSecond := url2Stat(newSecondURL)
		switch {
		case errFirst == nil && errSecond != nil:
			ch <- DiffMessage{
				FirstURL:  newFirstURL,
				SecondURL: newSecondURL,
				Diff:      "only-in-first",
			}
			continue
		case errFirst == nil && errSecond == nil:
			switch {
			case newFirstContent.Type.IsDir():
				if !newSecondContent.Type.IsDir() {
					ch <- DiffMessage{
						FirstURL:  newFirstURL,
						SecondURL: newSecondURL,
						Diff:      "type",
					}
				}
				continue
			case newFirstContent.Type.IsRegular():
				if !newSecondContent.Type.IsRegular() {
					ch <- DiffMessage{
						FirstURL:  newFirstURL,
						SecondURL: newSecondURL,
						Diff:      "type",
					}
					continue
				}
				doDiffObjects(newFirstURL, newSecondURL, ch)
			}
		}
	} // End of for-loop
}

func dodiffRecursive(firstClnt, secondClnt client.Client, ch chan DiffMessage) {
	firstURLDelimited := firstClnt.URL().String()
	secondURLDelimited := secondClnt.URL().String()
	if strings.HasSuffix(firstURLDelimited, "/") == false {
		firstURLDelimited = firstURLDelimited + "/"
	}
	if strings.HasSuffix(secondURLDelimited, "/") == false {
		secondURLDelimited = secondURLDelimited + "/"
	}
	firstClnt, err := url2Client(firstURLDelimited)
	if err != nil {
		ch <- DiffMessage{Error: err.Trace()}
		return
	}
	secondClnt, err = url2Client(secondURLDelimited)
	if err != nil {
		ch <- DiffMessage{Error: err.Trace()}
		return
	}

	fch := firstClnt.List(true, false)
	sch := secondClnt.List(true, false)
	f, fok := <-fch
	s, sok := <-sch
	for {
		if fok == false {
			break
		}
		if f.Err != nil {
			ch <- DiffMessage{Error: f.Err.Trace()}
			continue
		}
		if f.Content.Type.IsDir() {
			// skip directories
			// there is no concept of directories on S3
			f, fok = <-fch
			continue
		}
		firstURL := firstURLDelimited + f.Content.Name
		secondURL := secondURLDelimited + f.Content.Name
		if sok == false {
			// Second list reached EOF
			ch <- DiffMessage{
				FirstURL:  firstURL,
				SecondURL: secondURL,
				Diff:      "only-in-first",
			}
			f, fok = <-fch
			continue
		}
		if s.Err != nil {
			ch <- DiffMessage{Error: s.Err.Trace()}
			continue
		}
		if s.Content.Type.IsDir() {
			// skip directories
			s, sok = <-sch
			continue
		}
		fC := f.Content
		sC := s.Content
		compare := strings.Compare(fC.Name, sC.Name)

		if compare == 0 {
			if fC.Type.IsRegular() {
				if !sC.Type.IsRegular() {
					ch <- DiffMessage{
						FirstURL:  firstURL,
						SecondURL: secondURL,
						Diff:      "type",
					}
				}
			} else if fC.Type.IsDir() {
				if !sC.Type.IsDir() {
					ch <- DiffMessage{
						FirstURL:  firstURL,
						SecondURL: secondURL,
						Diff:      "type",
					}
				}
			} else if fC.Size != sC.Size {
				ch <- DiffMessage{
					FirstURL:  firstURL,
					SecondURL: secondURL,
					Diff:      "size",
				}
			}
			f, fok = <-fch
			s, sok = <-sch
		}
		if compare < 0 {
			ch <- DiffMessage{
				FirstURL:  firstURL,
				SecondURL: secondURL,
				Diff:      "only-in-first",
			}
			f, fok = <-fch
		}
		if compare > 0 {
			s, sok = <-sch
		}
	}
}

// doDiffDirs - Diff two Dir URLs
func doDiffDirs(firstClnt, secondClnt client.Client, recursive bool, ch chan DiffMessage) {
	if recursive {
		dodiffRecursive(firstClnt, secondClnt, ch)
		return
	}
	dodiff(firstClnt, secondClnt, ch)
}
