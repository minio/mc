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
	srcContent, err := url2Content(newSrcURL)
	fatalIf(err.Trace(srcURL), "Unable to stat source ‘"+newSrcURL+"’.")

	if !srcContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(srcContent.URL.String(), srcContent.Type.String()), fmt.Sprintf("Source ‘%s’ is not a folder. Only folders are supported by mirror.", srcURL))
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
		if content != nil {
			if !content.Type.IsDir() {
				fatalIf(errInvalidArgument().Trace(), "Target ‘"+tgtURL+"’ is not a folder.")
			}
		}
	}
}

func getContent(ch <-chan client.ContentOnChannel) (c *client.Content) {
	for rv := range ch {
		if rv.Err != nil {
			continue
		}
		if rv.Content.Type.IsDir() {
			// ignore directories.
			continue
		}
		c = rv.Content
		break
	}
	return
}

func getTargetContent(srcContent *client.Content, targetContent *client.Content, targetCh <-chan client.ContentOnChannel, targetClnt client.Client) (c *client.Content) {
	if srcContent == nil {
		// nothing to do for empty source content.
		return
	}

	if targetContent == nil {
		c = getContent(targetCh)
	} else {
		c = targetContent
	}

	for ; c != nil; c = getContent(targetCh) {
		// Remove prefix so that we can properly validate.
		targetURL := strings.TrimPrefix(c.URL.Path, targetClnt.GetURL().Path)
		sourceURL := strings.TrimPrefix(srcContent.URL.Path, string(srcContent.URL.Separator))
		if sourceURL <= targetURL {
			break
		}
	}
	return
}

func deltaSourceTargets(sourceURL string, targetURLs []string, mirrorURLsCh chan<- mirrorURLs) {
	defer close(mirrorURLsCh)

	newSourceURL := stripRecursiveURL(sourceURL)
	if strings.HasSuffix(newSourceURL, "/") == false {
		newSourceURL = newSourceURL + "/"
	}

	sourceClient, err := url2Client(newSourceURL)
	if err != nil {
		mirrorURLsCh <- mirrorURLs{Error: err.Trace(sourceURL)}
		return
	}

	targetLen := len(targetURLs)
	newTargetURLs := make([]string, targetLen)
	targetClients := make([]client.Client, targetLen)
	for i, targetURL := range targetURLs {
		targetClient, targetContent, err := url2Stat(targetURL)
		if err != nil {
			mirrorURLsCh <- mirrorURLs{Error: err.Trace(targetURL)}
			return
		}
		// targets have to be directory.
		if !targetContent.Type.IsDir() {
			mirrorURLsCh <- mirrorURLs{Error: errInvalidTarget(targetURL).Trace(newSourceURL)}
			return
		}
		// special case, be extremely careful before changing this behavior - will lead to data loss.
		newTargetURL := strings.TrimSuffix(targetURL, string(targetClient.GetURL().Separator)) + string(targetClient.GetURL().Separator)
		targetClient, err = url2Client(newTargetURL)
		if err != nil {
			mirrorURLsCh <- mirrorURLs{Error: err.Trace(newTargetURL)}
			return
		}
		targetClients[i] = targetClient
		newTargetURLs[i] = newTargetURL
	}

	targetChs := make([]<-chan client.ContentOnChannel, targetLen)
	for i, targetClient := range targetClients {
		targetChs[i] = targetClient.List(true, false)
	}
	targetContents := make([]*client.Content, targetLen)

	srcCh := sourceClient.List(true, false)
	for srcContent := getContent(srcCh); srcContent != nil; srcContent = getContent(srcCh) {
		var mirrorTargets []*client.Content
		for i := range targetChs {
			targetContents[i] = getTargetContent(srcContent, targetContents[i], targetChs[i], targetClients[i])

			// Remove prefix so that we can properly validate.
			var newTargetURLPath string
			if targetContents[i] != nil {
				newTargetURLPath = strings.TrimPrefix(targetContents[i].URL.Path, targetClients[i].GetURL().Path)
			}
			// Trim any preceding separators.
			newSourceURLPath := strings.TrimPrefix(srcContent.URL.Path, string(srcContent.URL.Separator))
			// either target reached EOF or target does not have source content.
			if targetContents[i] == nil || newSourceURLPath != newTargetURLPath {
				targetURL := client.NewURL(urlJoinPath(newTargetURLs[i], srcContent.URL.String()))
				mirrorTargets = append(mirrorTargets, &client.Content{URL: *targetURL})
				continue
			}

			if newSourceURLPath == newTargetURLPath {
				// source and target have same content type.
				if srcContent.Type.IsRegular() && targetContents[i].Type.IsRegular() {
					if srcContent.Size != targetContents[i].Size {
						if !mirrorIsForce {
							mirrorURLsCh <- mirrorURLs{
								Error: errOverWriteNotAllowed(targetContents[i].URL.String()).Trace(srcContent.URL.String()),
							}
							continue
						}
						targetURL := client.NewURL(urlJoinPath(newTargetURLs[i], srcContent.URL.String()))
						mirrorTargets = append(mirrorTargets, &client.Content{URL: *targetURL})
					}
					continue
				}
				if srcContent.Type.IsRegular() && !targetContents[i].Type.IsRegular() {
					// but type mismatches, error condition fail and continue.
					mirrorURLsCh <- mirrorURLs{
						Error: errInvalidTarget(targetContents[i].URL.String()).Trace(srcContent.URL.String()),
					}
					continue
				}
			}
		}
		if len(mirrorTargets) > 0 {
			mirrorURLsCh <- mirrorURLs{
				SourceContent:  srcContent,
				TargetContents: mirrorTargets,
			}
		}
	}
}

func prepareMirrorURLs(sourceURL string, targetURLs []string) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)
	go deltaSourceTargets(sourceURL, targetURLs, mirrorURLsCh)
	return mirrorURLsCh
}
