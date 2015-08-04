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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
	"github.com/tchap/go-patricia/patricia"
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
	Fatal(err)

	srcURL := URLs[0]
	tgtURLs := URLs[1:]

	/****** Generic rules *******/
	// Source cannot be a folder (except when recursive)
	if !isURLRecursive(srcURL) {
		_, srcContent, err := url2Stat(srcURL)
		Fatal(err)

		if !srcContent.Type.IsRegular() {
			if srcContent.Type.IsDir() {
				console.Fatalf("Source ‘%s’ is a folder. Please use ‘%s...’ to recursively copy this folder and its contents.\n", srcURL, srcURL)
			}
			console.Fatalf("Source ‘%s’ is not a regular file.\n", srcURL)
		}
	}

	for _, tgtURL := range tgtURLs {
		url, err := client.Parse(tgtURL)
		if err != nil {
			console.Fatalf("Unable to parse target ‘%s’ argument. %s\n", tgtURL, err)
		}
		if url.Host != "" {
			if url.Path == string(url.Separator) {
				console.Fatalf("Bucket creation detected for %s, cloud storage URL's should use ‘mc mb’ to create buckets\n", tgtURL)
			}
		}
	}

	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to mirror command. %s\n", errInvalidArgument{})
	}
	srcURL = stripRecursiveURL(srcURL)
	_, srcContent, err := url2Stat(srcURL)
	Fatal(err)

	if srcContent.Type.IsRegular() { // Ellipses is supported only for folders.
		Fatal(probe.New(errSourceIsNotDir{URL: srcURL}))
	}
	for _, tgtURL := range tgtURLs {
		_, content, err := url2Stat(tgtURL)
		if err == nil {
			if !content.Type.IsDir() {
				Fatal(probe.New(errTargetIsNotDir{URL: tgtURL}))
			}
		}
	}
}

func deltaSourceTargets(sourceClnt client.Client, targetClnts []client.Client) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)
	go func() {
		defer close(mirrorURLsCh)
		sourceTrie := patricia.NewTrie()
		targetTries := make([]*patricia.Trie, len(targetClnts))
		wg := new(sync.WaitGroup)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for sourceContentCh := range sourceClnt.List(true) {
				if sourceContentCh.Err != nil {
					mirrorURLsCh <- mirrorURLs{Error: sourceContentCh.Err.Trace()}
					return
				}
				if sourceContentCh.Content.Type.IsRegular() {
					sourceTrie.Insert(patricia.Prefix(sourceContentCh.Content.Name), sourceContentCh.Content.Size)
				}
			}
		}()

		for i, targetClnt := range targetClnts {
			wg.Add(1)
			go func(i int, targetClnt client.Client) {
				defer wg.Done()
				targetTrie := patricia.NewTrie()
				for targetContentCh := range targetClnt.List(true) {
					if targetContentCh.Err != nil {
						mirrorURLsCh <- mirrorURLs{Error: targetContentCh.Err.Trace()}
						return
					}
					if targetContentCh.Content.Type.IsRegular() {
						targetTrie.Insert(patricia.Prefix(targetContentCh.Content.Name), struct{}{})
					}
				}
				targetTries[i] = targetTrie
			}(i, targetClnt)
		}
		wg.Wait()

		matchNameCh := make(chan string, 10000)
		go func(matchNameCh chan<- string) {
			itemFunc := func(prefix patricia.Prefix, item patricia.Item) error {
				matchNameCh <- string(prefix)
				return nil
			}
			sourceTrie.Visit(itemFunc)
			defer close(matchNameCh)
		}(matchNameCh)
		for matchName := range matchNameCh {
			sourceContent := new(client.Content)
			var targetContents []*client.Content
			for i, targetTrie := range targetTries {
				if !targetTrie.Match(patricia.Prefix(matchName)) {
					sourceURLDelimited := sourceClnt.URL().String()[:strings.LastIndex(sourceClnt.URL().String(),
						string(sourceClnt.URL().Separator))+1]
					newTargetURLParse := *targetClnts[i].URL()
					newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, matchName)
					sourceContent.Size = sourceTrie.Get(patricia.Prefix(matchName)).(int64)
					sourceContent.Name = sourceURLDelimited + matchName
					targetContents = append(targetContents, &client.Content{Name: newTargetURLParse.String()})
				}
			}
			mirrorURLsCh <- mirrorURLs{
				SourceContent:  sourceContent,
				TargetContents: targetContents,
			}
		}
	}()
	return mirrorURLsCh
}

// prepareMirrorURLs - prepares target and source URLs for mirroring.
func prepareMirrorURLs(sourceURL string, targetURLs []string) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)

	go func() {
		defer close(mirrorURLsCh)
		sourceClnt, err := url2Client(stripRecursiveURL(sourceURL))
		if err != nil {
			mirrorURLsCh <- mirrorURLs{Error: err.Trace()}
			return
		}
		targetClnts := make([]client.Client, len(targetURLs))
		for i, targetURL := range targetURLs {
			targetURL = stripRecursiveURL(targetURL)
			targetClnt, targetContent, err := url2Stat(targetURL)
			if err != nil {
				mirrorURLsCh <- mirrorURLs{Error: err.Trace()}
				return
			}
			// if one of the targets is not dir exit
			if !targetContent.Type.IsDir() {
				mirrorURLsCh <- mirrorURLs{Error: probe.New(errInvalidTarget{URL: targetURL})}
				return
			}
			// special case, be extremely careful before changing this behavior - will lead to data loss
			newTargetURL := strings.TrimSuffix(targetURL, string(targetClnt.URL().Separator)) + string(targetClnt.URL().Separator)
			targetClnt, err = url2Client(newTargetURL)
			if err != nil {
				mirrorURLsCh <- mirrorURLs{Error: err.Trace()}
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
