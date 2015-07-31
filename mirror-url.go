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
	"github.com/minio/minio/pkg/iodine"
	"github.com/tchap/go-patricia/patricia"
)

type mirrorURLs struct {
	SourceContent  *client.Content
	TargetContents []*client.Content
	Error          error `json:"-"`
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
	if err != nil {
		console.Fatalf("One or more unknown URL types found %s. %s\n", ctx.Args(), NewIodine(iodine.New(err, nil)))
	}

	srcURL := URLs[0]
	tgtURLs := URLs[1:]

	/****** Generic rules *******/
	// Source cannot be a folder (except when recursive)
	if !isURLRecursive(srcURL) {
		_, srcContent, err := url2Stat(srcURL)
		// Source exist?.
		if err != nil {
			console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, NewIodine(iodine.New(err, nil)))
		}
		if !srcContent.Type.IsRegular() {
			if srcContent.Type.IsDir() {
				console.Fatalf("Source ‘%s’ is a folder. Please use ‘%s...’ to recursively copy this folder and its contents.\n", srcURL, srcURL)
			}
			console.Fatalf("Source ‘%s’ is not a regular file.\n", srcURL)
		}
	}
	// Recursive URLs are not allowed in target.
	for _, tgtURL := range tgtURLs {
		if isURLRecursive(tgtURL) {
			console.Fatalf("Target ‘%s’ cannot be recursive. %s\n", tgtURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
		}
	}

	for _, tgtURL := range tgtURLs {
		url, err := client.Parse(tgtURL)
		if err != nil {
			console.Fatalf("Unable to parse target ‘%s’ argument. %s\n", tgtURL, NewIodine(iodine.New(err, nil)))
		}
		if url.Host != "" {
			if url.Path == string(url.Separator) {
				console.Fatalf("Bucket creation detected for %s, cloud storage URL's should use ‘mc mb’ to create buckets\n", tgtURL)
			}
		}
	}

	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to mirror command. %s\n", NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	srcURL = stripRecursiveURL(srcURL)
	_, srcContent, err := url2Stat(srcURL)
	// Source exist?.
	if err != nil {
		console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, NewIodine(iodine.New(err, nil)))
	}

	if srcContent.Type.IsRegular() { // Ellipses is supported only for folders.
		console.Fatalf("Source ‘%s’ is not a folder. %s\n", stripRecursiveURL(srcURL), NewIodine(iodine.New(err, nil)))
	}
	for _, tgtURL := range tgtURLs {
		_, content, err := url2Stat(tgtURL)
		if err == nil {
			if !content.Type.IsDir() {
				console.Fatalf("One of the target ‘%s’ is not a folder. cannot have mixtures of directories and files while copying directories recursively. %s\n", tgtURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
			}
		}
	}
}

// prepareSingleMirrorURLTypeA - prepares a single source and single target argument for mirroring.
func prepareSingleMirrorURLsTypeA(sourceURL string, targetURL string) mirrorURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil { // Source does not exist or insufficient privileges.
		return mirrorURLs{Error: NewIodine(iodine.New(err, nil))}
	}
	if !sourceContent.Type.IsRegular() { // Source is not a regular file
		return mirrorURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	// All OK.. We can proceed. Type A
	sourceContent.Name = sourceURL
	return mirrorURLs{SourceContent: sourceContent, TargetContents: []*client.Content{{Name: targetURL}}}
}

// prepareMirrorURLsTypeA - A: mirror(f, f) -> mirror(f, f)
func prepareMirrorURLsTypeA(sourceURL string, targetURLs []string) mirrorURLs {
	var sURLs mirrorURLs
	for _, targetURL := range targetURLs { // Prepare each target separately
		URLs := prepareSingleMirrorURLsTypeA(sourceURL, targetURL)
		if URLs.Error != nil {
			return mirrorURLs{Error: NewIodine(iodine.New(URLs.Error, nil))}
		}
		sURLs.SourceContent = URLs.SourceContent
		sURLs.TargetContents = append(sURLs.TargetContents, URLs.TargetContents...)
	}
	return sURLs
}

func deltaSourceTargets(sourceClnt client.Client, targetClnts []client.Client) {
	sourceTrie := patricia.NewTrie()
	var targetTries []*patricia.Trie
	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for sourceContentCh := range sourceClnt.List(true) {
			if sourceContentCh.Err != nil {
				return
			}
			sourceURLDelimited := sourceClnt.URL().String()[:strings.LastIndex(sourceClnt.URL().String(), string(sourceClnt.URL().Separator))+1]
			newSourceURL := sourceURLDelimited + sourceContentCh.Content.Name
			newSourceURLParse, err := client.Parse(newSourceURL)
			if err != nil {
				return
			}
			sourceTrie.Insert(patricia.Prefix(newSourceURLParse.String()), struct{}{})
		}
	}()

	for _, targetClnt := range targetClnts {
		wg.Add(1)
		go func(targetClnt client.Client) {
			defer wg.Done()
			targetTrie := patricia.NewTrie()
			for targetContentCh := range targetClnt.List(true) {
				if targetContentCh.Err != nil {
					return
				}
				targetURLDelimited := targetClnt.URL().String()[:strings.LastIndex(targetClnt.URL().String(), string(targetClnt.URL().Separator))+1]
				newTargetURL := targetURLDelimited + targetContentCh.Content.Name
				newTargetURLParse, err := client.Parse(newTargetURL)
				if err != nil {
					return
				}
				targetTrie.Insert(patricia.Prefix(newTargetURLParse.String()), struct{}{})
				targetTries = append(targetTries, targetTrie)
			}
		}(targetClnt)
	}
	wg.Wait()

	matchURLCh := make(chan string, 10000)
	go func(matchURLCh chan<- string) {
		itemFunc := func(prefix patricia.Prefix, item patricia.Item) error {
			matchURLCh <- string(prefix)
			return nil
		}
		sourceTrie.Visit(itemFunc)
		defer close(matchURLCh)
	}(matchURLCh)
	for matchURL := range matchURLCh {
		for _, targetTrie := range targetTries {
			if targetTrie.Match(patricia.Prefix(matchURL)) {
				continue
			}
		}
	}
}

// prepareMirrorURLsTypeC - C:
func prepareMirrorURLsTypeC(sourceURL string, targetURLs []string) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)
	go func() {
		defer close(mirrorURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(errSourceNotRecursive{URL: sourceURL}, nil))}
			return
		}
		// add `/` after trimming off `...` to emulate folders
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, sourceContent, err := url2Stat(sourceURL)
		// Source exist?
		if err != nil {
			// Source does not exist or insufficient privileges.
			mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(errSourceIsNotDir{URL: sourceURL}, nil))}
			return
		}

		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(sourceContent.Err, nil))}
				continue
			}
			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for mirror.
				continue
			}
			// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
				continue
			}
			var newTargetURLs []string
			var sourceContentParse *client.URL
			for _, targetURL := range targetURLs {
				targetURLParse, err := client.Parse(targetURL)
				if err != nil {
					mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
					continue
				}
				sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
					string(sourceURLParse.Separator))+1]
				sourceContentName := sourceContent.Content.Name
				sourceContentURL := sourceURLDelimited + sourceContentName
				sourceContentParse, err = client.Parse(sourceContentURL)
				if err != nil {
					mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceContentName}, nil))}
					continue
				}
				// Construct target path from recursive path of source without its prefix dir.
				newTargetURLParse := *targetURLParse
				newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
				newTargetURLs = append(newTargetURLs, newTargetURLParse.String())
			}
			mirrorURLsCh <- prepareMirrorURLsTypeA(sourceContentParse.String(), newTargetURLs)
		}
	}()
	return mirrorURLsCh
}

// prepareMirrorURLs - prepares target and source URLs for mirroring.
func prepareMirrorURLs(sourceURL string, targetURLs []string) <-chan mirrorURLs {
	mirrorURLsCh := make(chan mirrorURLs)
	go func() {
		defer close(mirrorURLsCh)
		for sURLs := range prepareMirrorURLsTypeC(sourceURL, targetURLs) {
			mirrorURLsCh <- sURLs
		}
	}()
	return mirrorURLsCh
}
