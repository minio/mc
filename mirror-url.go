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
	SourceContent *client.Content
	TargetContent *client.Content
	Error         *probe.Error `json:"-"`
}

func (m mirrorURLs) isEmpty() bool {
	if m.SourceContent == nil && m.TargetContent == nil && m.Error == nil {
		return true
	}
	if m.SourceContent.Size == 0 && m.TargetContent == nil && m.Error == nil {
		return true
	}
	return false
}

//
//   * MIRROR ARGS - VALID CASES
//   =========================
//   mirror(d1..., d2) -> []mirror(d1/f, d2/d1/f)

// checkMirrorSyntax(URLs []string)
func checkMirrorSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "mirror", 1) // last argument is exit code.
	}

	// extract URLs.
	URLs, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	srcURL := URLs[0]
	tgtURL := URLs[1]

	/****** Generic rules *******/
	_, srcContent, err := url2Stat(srcURL)
	if err != nil && !isURLPrefixExists(srcURL) {
		fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+srcURL+"’.")
	}

	if err == nil && !srcContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(srcContent.URL.String(), srcContent.Type.String()), fmt.Sprintf("Source ‘%s’ is not a folder. Only folders are supported by mirror command.", srcURL))
	}

	if len(tgtURL) == 0 && tgtURL == "" {
		fatalIf(errInvalidArgument().Trace(), "Invalid target arguments to mirror command.")
	}

	url := client.NewURL(tgtURL)
	if url.Host != "" {
		if url.Path == string(url.Separator) {
			fatalIf(errInvalidArgument().Trace(tgtURL),
				fmt.Sprintf("Target ‘%s’ does not contain bucket name.", tgtURL))
		}
	}
	_, _, err = url2Stat(tgtURL)
	// we die on any error other than client.PathNotFound - destination directory need not exist.
	if _, ok := err.ToGoError().(client.PathNotFound); !ok {
		fatalIf(err.Trace(tgtURL), fmt.Sprintf("Unable to stat %s", tgtURL))
	}
}

func deltaSourceTargets(sourceURL string, targetURL string, isForce bool, mirrorURLsCh chan<- mirrorURLs) {
	defer close(mirrorURLsCh)

	// source and targets are always directories
	sourceSeparator := string(client.NewURL(sourceURL).Separator)
	if !strings.HasSuffix(sourceURL, sourceSeparator) {
		sourceURL = sourceURL + sourceSeparator
	}
	targetSeparator := string(client.NewURL(targetURL).Separator)
	if !strings.HasSuffix(targetURL, targetSeparator) {
		targetURL = targetURL + targetSeparator
	}

	objectDifferenceTarget, err := objectDifferenceFactory(targetURL)
	if err != nil {
		mirrorURLsCh <- mirrorURLs{Error: err.Trace(targetURL)}
		return
	}

	sourceClient, err := url2Client(sourceURL)
	if err != nil {
		mirrorURLsCh <- mirrorURLs{Error: err.Trace(sourceURL)}
		return
	}

	for sourceContent := range sourceClient.List(true, false) {
		if sourceContent.Err != nil {
			mirrorURLsCh <- mirrorURLs{Error: sourceContent.Err.Trace(sourceClient.GetURL().String())}
			continue
		}
		if sourceContent.Type.IsDir() {
			continue
		}
		suffix := strings.TrimPrefix(sourceContent.URL.String(), sourceURL)
		differ, err := objectDifferenceTarget(suffix, sourceContent.Type, sourceContent.Size)
		if err != nil {
			mirrorURLsCh <- mirrorURLs{Error: err.Trace(sourceContent.URL.String())}
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
		if differ == differSize && !isForce {
			// size differs and force not set
			mirrorURLsCh <- mirrorURLs{Error: errOverWriteNotAllowed(sourceContent.URL.String())}
			continue
		}
		// either available only in source or size differs and force is set
		targetPath := urlJoinPath(targetURL, suffix)
		targetContent := &client.Content{URL: *client.NewURL(targetPath)}
		mirrorURLsCh <- mirrorURLs{
			SourceContent: sourceContent,
			TargetContent: targetContent,
		}
	}
}

func prepareMirrorURLs(sourceURL string, targetURL string, isForce bool) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)
	go deltaSourceTargets(sourceURL, targetURL, isForce, mirrorURLsCh)
	return mirrorURLsCh
}
