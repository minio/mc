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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

type mirrorURLs struct {
	SourceContent  *client.Content
	TargetContents []*client.Content
	Error          error `json:"-"`
}

type mirrorURLsType uint8

const (
	mirrorURLsTypeInvalid mirrorURLsType = iota
	mirrorURLsTypeA
	mirrorURLsTypeB
	mirrorURLsTypeC
)

//   NOTE: All the parse rules should reduced to A: Mirror(Source, []Target).
//
//   * CAST ARGS - VALID CASES
//   =========================
//   A: mirror(f, []f) -> mirror(f, []f)
//   B: mirror(f, [](d | f)) -> mirror(f, [](d/f | f)) -> A:
//   C: mirror(d1..., [](d2)) -> []mirror(d1/f, [](d2/d1/f)) -> []A:

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

	switch guessMirrorURLType(srcURL, tgtURLs) {
	case mirrorURLsTypeA: // File -> File.
		checkMirrorSyntaxTypeA(srcURL, tgtURLs)
	case mirrorURLsTypeB: // File -> Folder.
		checkMirrorSyntaxTypeB(srcURL, tgtURLs)
	case mirrorURLsTypeC: // Folder -> Folder.
		checkMirrorSyntaxTypeC(srcURL, tgtURLs)
	default:
		console.Fatalln("Invalid arguments. Unable to determine how to mirror. Please report this issue at https://github.com/minio/mc/issues")
	}
}

func checkMirrorSyntaxTypeA(srcURL string, tgtURLs []string) {
	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to mirror command. %s\n", NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	_, srcContent, err := url2Stat(srcURL)
	// Source exist?.
	if err != nil {
		console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, NewIodine(iodine.New(err, nil)))
	}
	if srcContent.Type.IsDir() {
		console.Fatalf("Source ‘%s’ is a folder. Use ‘%s...’ argument to mirror this folder and its contents recursively. %s\n", srcURL, srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	if !srcContent.Type.IsRegular() {
		console.Fatalf("Source ‘%s’ is not a file. %s\n", srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	for _, tgtURL := range tgtURLs {
		_, tgtContent, err := url2Stat(tgtURL)
		// Target exist?.
		if err == nil {
			if !tgtContent.Type.IsRegular() {
				console.Fatalf("Target ‘%s’ is not a file. %s\n", tgtURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
			}
		}
	}
}

func checkMirrorSyntaxTypeB(srcURL string, tgtURLs []string) {
	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to mirror command. %s\n", NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	_, srcContent, err := url2Stat(srcURL)
	// Source exist?.
	if err != nil {
		console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, NewIodine(iodine.New(err, nil)))
	}
	if srcContent.Type.IsDir() {
		console.Fatalf("Source ‘%s’ is a folder. Use ‘%s...’ argument to mirror this folder and its contents recursively. %s\n", srcURL, srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	if !srcContent.Type.IsRegular() {
		console.Fatalf("Source ‘%s’ is not a file. %s\n", srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	// targetURL can be folder or file, internally TypeB calls TypeA if it finds a file
}

func checkMirrorSyntaxTypeC(srcURL string, tgtURLs []string) {
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

// guessMirrorURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessMirrorURLType(sourceURL string, targetURLs []string) mirrorURLsType {
	if targetURLs == nil || len(targetURLs) == 0 { // Target is empty
		return mirrorURLsTypeInvalid
	}
	if sourceURL == "" { // Source is empty
		return mirrorURLsTypeInvalid
	}
	for _, targetURL := range targetURLs {
		if targetURL == "" { // One of the target is empty
			return mirrorURLsTypeInvalid
		}
	}

	if isURLRecursive(sourceURL) { // Type C
		return mirrorURLsTypeC
	} // else Type A or Type B
	for _, targetURL := range targetURLs {
		if isTargetURLDir(targetURL) { // Type B
			return mirrorURLsTypeB
		}
	} // else Type A
	return mirrorURLsTypeA
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

// prepareSingleMirrorURLsTypeB - prepares a single target and single source URLs for mirroring.
func prepareSingleMirrorURLsTypeB(sourceURL string, targetURL string) mirrorURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return mirrorURLs{Error: NewIodine(iodine.New(err, nil))}
	}

	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file.
		return mirrorURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	_, targetContent, err := url2Stat(targetURL)
	if err != nil {
		// Source and target are files. Already reduced to Type A.
		return prepareSingleMirrorURLsTypeA(sourceURL, targetURL)
	}

	if targetContent.Type.IsRegular() { // File to File
		// Source and target are files. Already reduced to Type A.
		return prepareSingleMirrorURLsTypeA(sourceURL, targetURL)
	}

	// Source is a file, target is a folder and exists.
	sourceURLParse, err := client.Parse(sourceURL)
	if err != nil {
		return mirrorURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return mirrorURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
	}
	// Reduce Type B to Type A.
	targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
	return prepareSingleMirrorURLsTypeA(sourceURL, targetURLParse.String())
}

// prepareMirrorURLsTypeB - B: mirror(f, d) -> mirror(f, d/f) -> A
func prepareMirrorURLsTypeB(sourceURL string, targetURLs []string) mirrorURLs {
	var sURLs mirrorURLs
	for _, targetURL := range targetURLs {
		URLs := prepareSingleMirrorURLsTypeB(sourceURL, targetURL)
		if URLs.Error != nil {
			return mirrorURLs{Error: NewIodine(iodine.New(URLs.Error, nil))}
		}
		sURLs.SourceContent = URLs.SourceContent
		sURLs.TargetContents = append(sURLs.TargetContents, URLs.TargetContents[0])
	}
	return sURLs
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
		switch guessMirrorURLType(sourceURL, targetURLs) {
		case mirrorURLsTypeA:
			mirrorURLsCh <- prepareMirrorURLsTypeA(sourceURL, targetURLs)
		case mirrorURLsTypeB:
			mirrorURLsCh <- prepareMirrorURLsTypeB(sourceURL, targetURLs)
		case mirrorURLsTypeC:
			for sURLs := range prepareMirrorURLsTypeC(sourceURL, targetURLs) {
				mirrorURLsCh <- sURLs
			}
		default:
			mirrorURLsCh <- mirrorURLs{Error: NewIodine(iodine.New(errInvalidArgument{}, nil))}
		}
	}()
	return mirrorURLsCh
}
