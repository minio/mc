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
	"sync"
	"time"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
	"github.com/tchap/go-patricia/patricia"
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

func mustURLJoinPath(url1, url2 string) string {
	newURL, _ := urlJoinPath(url1, url2)
	return newURL
}

// urlJoinPath Join a path to existing URL
func urlJoinPath(url1, url2 string) (newURLStr string, err error) {
	u1, err := client.Parse(url1)
	if err != nil {
		return "", NewIodine(iodine.New(err, nil))
	}
	u2, err := client.Parse(url2)
	if err != nil {
		return "", NewIodine(iodine.New(err, nil))
	}
	u1.Path = filepath.Join(u1.Path, u2.Path)
	newURLStr = u1.String()
	return newURLStr, nil
}

func doDiffInRoutine(firstURL, secondURL string, recursive bool, ch chan diff) {
	defer close(ch)
	firstClnt, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + firstURL + "’",
			err:     NewIodine(iodine.New(err, nil)),
		}
		return
	}
	secondClnt, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + secondURL + "’",
			err:     NewIodine(iodine.New(err, nil)),
		}
		return
	}
	if firstContent.Type.IsRegular() {
		switch {
		case secondContent.Type.IsDir():
			newSecondURL, err := urlJoinPath(secondURL, firstURL)
			if err != nil {
				ch <- diff{
					message: "Unable to construct new URL from ‘" + secondURL + "’ using ‘" + firstURL,
					err:     NewIodine(iodine.New(err, nil)),
				}
				return
			}
			doDiffObjects(firstURL, newSecondURL, ch)
		case !secondContent.Type.IsRegular():
			ch <- diff{
				message: "‘" + firstURL + "’ and " + "‘" + secondURL + "’ differs in type.",
				err:     nil,
			}
			return
		case secondContent.Type.IsRegular():
			doDiffObjects(firstURL, secondURL, ch)
		}
	}
	if firstContent.Type.IsDir() {
		switch {
		case !secondContent.Type.IsDir():
			ch <- diff{
				message: "‘" + firstURL + "’ and " + "‘" + secondURL + "’ differs in type.",
				err:     nil,
			}
			return
		default:
			doDiffDirs(firstClnt, secondClnt, recursive, ch)
		}
	}
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
			err:     NewIodine(iodine.New(errNotAnObject{url: firstURL}, nil)),
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

func dodiff(firstClnt, secondClnt client.Client, ch chan diff) {
	for contentCh := range firstClnt.List(false) {
		if contentCh.Err != nil {
			ch <- diff{
				message: "Failed to list ‘" + firstClnt.URL().String() + "’",
				err:     NewIodine(iodine.New(contentCh.Err, nil)),
			}
			return
		}
		newFirstURL, err := urlJoinPath(firstClnt.URL().String(), contentCh.Content.Name)
		if err != nil {
			ch <- diff{
				message: "Unable to construct new URL from ‘" + firstClnt.URL().String() + "’ using ‘" + contentCh.Content.Name + "’",
				err:     NewIodine(iodine.New(err, nil)),
			}
			return
		}
		newSecondURL, err := urlJoinPath(secondClnt.URL().String(), contentCh.Content.Name)
		if err != nil {
			ch <- diff{
				message: "Unable to construct new URL from ‘" + secondClnt.URL().String() + "’ using ‘" + contentCh.Content.Name + "’",
				err:     NewIodine(iodine.New(err, nil)),
			}
			return
		}
		_, newFirstContent, errFirst := url2Stat(newFirstURL)
		_, newSecondContent, errSecond := url2Stat(newSecondURL)
		switch {
		case errFirst != nil && errSecond == nil:
			ch <- diff{
				message: "‘" + newSecondURL + "’ Only in ‘" + secondClnt.URL().String() + "’",
				err:     nil,
			}
			continue
		case errFirst == nil && errSecond != nil:
			ch <- diff{
				message: "‘" + newFirstURL + "’ Only in ‘" + firstClnt.URL().String() + "’",
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

func dodiffRecursive(firstClnt, secondClnt client.Client, ch chan diff) {
	firstTrie := patricia.NewTrie()
	secondTrie := patricia.NewTrie()
	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func(ch chan<- diff) {
		defer wg.Done()
		for firstContentCh := range firstClnt.List(true) {
			if firstContentCh.Err != nil {
				ch <- diff{
					message: "Failed to list ‘" + firstClnt.URL().String() + "’",
					err:     NewIodine(iodine.New(firstContentCh.Err, nil)),
				}
				return
			}
			firstTrie.Insert(patricia.Prefix(firstContentCh.Content.Name), struct{}{})
		}
	}(ch)
	wg.Add(1)
	go func(ch chan<- diff) {
		defer wg.Done()
		for secondContentCh := range secondClnt.List(true) {
			if secondContentCh.Err != nil {
				ch <- diff{
					message: "Failed to list ‘" + secondClnt.URL().String() + "’",
					err:     NewIodine(iodine.New(secondContentCh.Err, nil)),
				}
				return
			}
			secondTrie.Insert(patricia.Prefix(secondContentCh.Content.Name), struct{}{})
		}
	}(ch)

	doneCh := make(chan struct{})
	defer close(doneCh)
	go func(doneCh <-chan struct{}) {
		cursorCh := cursorAnimate()
		for {
			select {
			case <-time.Tick(100 * time.Millisecond):
				console.PrintC("\r" + "Scanning.. " + string(<-cursorCh))
			case <-doneCh:
				return
			}
		}
	}(doneCh)
	wg.Wait()
	doneCh <- struct{}{}
	console.PrintC("\r" + "Finished" + "\n")

	matchNameCh := make(chan string, 10000)
	go func(matchNameCh chan<- string) {
		itemFunc := func(prefix patricia.Prefix, item patricia.Item) error {
			matchNameCh <- string(prefix)
			return nil
		}
		firstTrie.Visit(itemFunc)
		defer close(matchNameCh)
	}(matchNameCh)
	for matchName := range matchNameCh {
		if !secondTrie.Match(patricia.Prefix(matchName)) {
			firstURLDelimited := firstClnt.URL().String()[:strings.LastIndex(firstClnt.URL().String(), string(firstClnt.URL().Separator))+1]
			firstURL := firstURLDelimited + matchName
			ch <- diff{
				message: "‘" + firstURL + "’ Only in ‘" + firstClnt.URL().String() + "’",
				err:     nil,
			}
		}
	}
}

// doDiffDirs - Diff two Dir URLs
func doDiffDirs(firstClnt, secondClnt client.Client, recursive bool, ch chan diff) {
	if recursive {
		dodiffRecursive(firstClnt, secondClnt, ch)
		return
	}
	dodiff(firstClnt, secondClnt, ch)
}
