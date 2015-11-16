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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
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
	if len(ctx.Args()) < 2 {
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
	if err != nil && !prefixExists(newSrcURL) {
		fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+newSrcURL+"’.")
	}

	if err == nil && !srcContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(srcContent.URL.String(), srcContent.Type.String()), fmt.Sprintf("Source ‘%s’ is not a folder. Only folders are supported by mirror command.", srcURL))
	}

	if len(tgtURLs) == 0 && tgtURLs == nil {
		fatalIf(errInvalidArgument().Trace(), "Invalid target arguments to mirror command.")
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
		if content != nil {
			if !content.Type.IsDir() {
				fatalIf(errInvalidArgument().Trace(), "Target ‘"+tgtURL+"’ is not a folder.")
			}
		}
	}
}

func deltaSourceTargets(sourceURL string, targetURLs []string, mirrorURLsCh chan<- mirrorURLs) {
	defer close(mirrorURLsCh)

	// source and targets are always directories
	sourceSeparator := string(client.NewURL(sourceURL).Separator)
	if !strings.HasSuffix(sourceURL, sourceSeparator) {
		sourceURL = sourceURL + sourceSeparator
	}
	for i, url := range targetURLs {
		targetSeparator := string(client.NewURL(url).Separator)
		if !strings.HasSuffix(url, targetSeparator) {
			targetURLs[i] = url + targetSeparator
		}
	}

	// array of objectDifference functions corresponding to their targetURL
	objectDifferenceArray := make([]objectDifference, len(targetURLs))

	for i := range targetURLs {
		var err *probe.Error
		objectDifferenceArray[i], err = objectDifferenceFactory(targetURLs[i])
		if err != nil {
			mirrorURLsCh <- mirrorURLs{Error: err.Trace()}
			return
		}
	}

	sourceClient, err := url2Client(sourceURL)
	if err != nil {
		mirrorURLsCh <- mirrorURLs{Error: err.Trace(sourceURL)}
		return
	}

	for sourceContent := range sourceClient.List(true, false) {
		if sourceContent.Err != nil {
			mirrorURLsCh <- mirrorURLs{Error: sourceContent.Err.Trace()}
			continue
		}
		if sourceContent.Content.Type.IsDir() {
			continue
		}
		suffix := strings.TrimPrefix(sourceContent.Content.URL.String(), sourceURL)
		targetContents := []*client.Content{}
		for i, difference := range objectDifferenceArray {
			differ, err := difference(suffix, sourceContent.Content.Type, sourceContent.Content.Size)
			if err != nil {
				mirrorURLsCh <- mirrorURLs{Error: err.Trace()}
				continue
			}
			if differ == differNone {
				// no difference, continue
				continue
			}
			if differ == differType {
				mirrorURLsCh <- mirrorURLs{Error: errInvalidTarget(suffix)}
				continue
			}
			if differ == differSize && !mirrorIsForce {
				// size differs and force not set
				mirrorURLsCh <- mirrorURLs{Error: errOverWriteNotAllowed(sourceContent.Content.URL.String())}
				continue
			}
			// either available only in source or size differs and force is set
			targetPath := urlJoinPath(targetURLs[i], suffix)
			targetContent := client.Content{URL: *client.NewURL(targetPath)}
			targetContents = append(targetContents, &targetContent)
		}
		if len(targetContents) > 0 {
			mirrorURLsCh <- mirrorURLs{
				SourceContent:  sourceContent.Content,
				TargetContents: targetContents,
			}
		}
	}
}

func prepareMirrorURLs(sourceURL string, targetURLs []string) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)
	go deltaSourceTargets(sourceURL, targetURLs, mirrorURLsCh)
	return mirrorURLsCh
}
