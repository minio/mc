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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

type mirrorURLs struct {
	SourceContent  *client.Content
	TargetContents []*client.Content
	Error          *probe.Error `json:"-"`
}

func (m mirrorURLs) isEmpty() bool {
	if m.SourceContent == nil && len(m.TargetContents) == 0 && m.Error == nil {
		return true
	}
	if m.SourceContent.Size == 0 && len(m.TargetContents) == 0 && m.Error == nil {
		return true
	}
	return false
}

//
//   * MIRROR ARGS - VALID CASES
//   =========================
//   mirror(d1..., [](d2)) -> []mirror(d1/f, [](d2/d1/f))

// checkMirrorSyntax(URLs []string)
func checkMirrorSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "mirror", 1) // last argument is exit code.
	}

	// extract URLs.
	URLs, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	srcURL := URLs[0]
	tgtURLs := URLs[1:]

	/****** Generic rules *******/
	// Recursive source URL.
	newSrcURL := stripRecursiveURL(srcURL)
	_, srcContent, err := url2Stat(newSrcURL)
	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+newSrcURL+"’.")

	if !srcContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(srcContent.Name, srcContent.Type.String()), fmt.Sprintf("Source ‘%s’ is not a folder. Only folders are supported by mirror.", srcURL))
	}

	if len(tgtURLs) == 0 && tgtURLs == nil {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of target arguments to mirror command.")
	}

	for _, tgtURL := range tgtURLs {
		// Recursive URLs are not allowed in target.
		if isURLRecursive(tgtURL) {
			fatalIf(errDummy().Trace(), fmt.Sprintf("Recursive option is not supported for target ‘%s’ argument.", tgtURL))
		}

		url := client.NewURL(tgtURL)
		if url.Host != "" {
			if url.Path == string(url.Separator) {
				fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Target ‘%s’ does not contain bucket name.", tgtURL))
			}
		}

		_, content, err := url2Stat(tgtURL)
		fatalIf(err.Trace(tgtURL), "Unable to stat target ‘"+tgtURL+"’.")
		if !content.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(), "Target ‘"+tgtURL+"’ is not a folder.")
		}
	}
}

func deltaSourceTargets(sourceClnt client.Client, targetClnts []client.Client) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)

	go func() {
		defer close(mirrorURLsCh)
		id := newRandomID(8)

		doneCh := make(chan bool)
		defer close(doneCh)
		go func(doneCh <-chan bool) {
			cursorCh := cursorAnimate()
			for {
				select {
				case <-time.Tick(100 * time.Millisecond):
					if !globalQuietFlag && !globalJSONFlag {
						console.PrintC("\r" + "Scanning.. " + string(<-cursorCh))
					}
				case <-doneCh:
					return
				}
			}
		}(doneCh)

		sourceSortedList := sortedList{}
		targetSortedList := make([]*sortedList, len(targetClnts))

		surldelimited := sourceClnt.URL().String()
		err := sourceSortedList.Create(sourceClnt, id+".src")
		if err != nil {
			mirrorURLsCh <- mirrorURLs{
				Error: err.Trace(),
			}
			return
		}

		turldelimited := make([]string, len(targetClnts))
		for i := range targetClnts {
			turldelimited[i] = targetClnts[i].URL().String()
			targetSortedList[i] = &sortedList{}
			err := targetSortedList[i].Create(targetClnts[i], id+"."+strconv.Itoa(i))
			if err != nil {
				// FIXME: do cleanup by calling Delete()
				mirrorURLsCh <- mirrorURLs{
					Error: err.Trace(),
				}
				return
			}
		}
		for source := range sourceSortedList.List(true) {
			if source.Content.Type.IsDir() {
				continue
			}
			targetContents := make([]*client.Content, 0, len(targetClnts))
			for i, t := range targetSortedList {
				match, err := t.Match(source.Content)
				if err != nil || match {
					// continue on io.EOF or if the keys matches
					// FIXME: handle other errors and ignore this target for future calls
					continue
				}
				targetContents = append(targetContents, &client.Content{Name: turldelimited[i] + source.Content.Name})
			}
			source.Content.Name = surldelimited + source.Content.Name
			if len(targetContents) > 0 {
				mirrorURLsCh <- mirrorURLs{
					SourceContent:  source.Content,
					TargetContents: targetContents,
				}
			}
		}
		if err := sourceSortedList.Delete(); err != nil {
			mirrorURLsCh <- mirrorURLs{
				Error: err.Trace(),
			}
		}
		for _, t := range targetSortedList {
			if err := t.Delete(); err != nil {
				mirrorURLsCh <- mirrorURLs{
					Error: err.Trace(),
				}
			}
		}
		doneCh <- true
		if !globalQuietFlag && !globalJSONFlag {
			console.Eraseline()
		}
	}()
	return mirrorURLsCh
}

// prepareMirrorURLs - prepares target and source URLs for mirroring.
func prepareMirrorURLs(sourceURL string, targetURLs []string) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)

	go func() {
		defer close(mirrorURLsCh)
		sourceURL = stripRecursiveURL(sourceURL)
		if strings.HasSuffix(sourceURL, "/") == false {
			sourceURL = sourceURL + "/"
		}
		sourceClnt, err := url2Client(sourceURL)
		if err != nil {
			mirrorURLsCh <- mirrorURLs{Error: err.Trace(sourceURL)}
			return
		}
		targetClnts := make([]client.Client, len(targetURLs))
		for i, targetURL := range targetURLs {
			targetClnt, targetContent, err := url2Stat(targetURL)
			if err != nil {
				mirrorURLsCh <- mirrorURLs{Error: err.Trace(targetURL)}
				return
			}
			// if one of the targets is not dir exit
			if !targetContent.Type.IsDir() {
				mirrorURLsCh <- mirrorURLs{Error: errInvalidTarget(targetURL).Trace()}
				return
			}
			// special case, be extremely careful before changing this behavior - will lead to data loss
			newTargetURL := strings.TrimSuffix(targetURL, string(targetClnt.URL().Separator)) + string(targetClnt.URL().Separator)
			targetClnt, err = url2Client(newTargetURL)
			if err != nil {
				mirrorURLsCh <- mirrorURLs{Error: err.Trace(newTargetURL)}
				return
			}
			targetClnts[i] = targetClnt
		}
		for sURLs := range deltaSourceTargets(sourceClnt, targetClnts) {
			mirrorURLsCh <- sURLs
		}
	}()
	return mirrorURLsCh
}
