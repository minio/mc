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

type castURLs struct {
	SourceContent  *client.Content
	TargetContents []*client.Content
	Error          error `json:"-"`
}

type castURLsType uint8

const (
	castURLsTypeInvalid castURLsType = iota
	castURLsTypeA
	castURLsTypeB
	castURLsTypeC
)

//   NOTE: All the parse rules should reduced to A: Cast(Source, []Target).
//
//   * CAST ARGS - VALID CASES
//   =========================
//   A: cast(f, []f) -> cast(f, []f)
//   B: cast(f, [](d | f)) -> cast(f, [](d/f | f)) -> A:
//   C: cast(d1..., [](d2)) -> []cast(d1/f, [](d2/d1/f)) -> []A:

// checkCastSyntax(URLs []string)
func checkCastSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cast", 1) // last argument is exit code.
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

	switch guessCastURLType(srcURL, tgtURLs) {
	case castURLsTypeA: // File -> File.
		checkCastSyntaxTypeA(srcURL, tgtURLs)
	case castURLsTypeB: // File -> Folder.
		checkCastSyntaxTypeB(srcURL, tgtURLs)
	case castURLsTypeC: // Folder -> Folder.
		checkCastSyntaxTypeC(srcURL, tgtURLs)
	default:
		console.Fatalln("Invalid arguments. Unable to determine how to cast. Please report this issue at https://github.com/minio/mc/issues")
	}
}

func checkCastSyntaxTypeA(srcURL string, tgtURLs []string) {
	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to cast command. %s\n", NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	_, srcContent, err := url2Stat(srcURL)
	// Source exist?.
	if err != nil {
		console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, NewIodine(iodine.New(err, nil)))
	}
	if srcContent.Type.IsDir() {
		console.Fatalf("Source ‘%s’ is a folder. Use ‘%s...’ argument to cast this folder and its contents recursively. %s\n", srcURL, srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
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

func checkCastSyntaxTypeB(srcURL string, tgtURLs []string) {
	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to cast command. %s\n", NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	_, srcContent, err := url2Stat(srcURL)
	// Source exist?.
	if err != nil {
		console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, NewIodine(iodine.New(err, nil)))
	}
	if srcContent.Type.IsDir() {
		console.Fatalf("Source ‘%s’ is a folder. Use ‘%s...’ argument to cast this folder and its contents recursively. %s\n", srcURL, srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	if !srcContent.Type.IsRegular() {
		console.Fatalf("Source ‘%s’ is not a file. %s\n", srcURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
	}
	// targetURL can be folder or file, internally TypeB calls TypeA if it finds a file
}

func checkCastSyntaxTypeC(srcURL string, tgtURLs []string) {
	if len(tgtURLs) == 0 && tgtURLs == nil {
		console.Fatalf("Invalid number of target arguments to cast command. %s\n", NewIodine(iodine.New(errInvalidArgument{}, nil)))
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
		if !isTargetURLDir(tgtURL) {
			console.Fatalf("One of the target ‘%s’ is not a folder. cannot have mixtures of directories and files while copying directories. %s\n", tgtURL, NewIodine(iodine.New(errInvalidArgument{}, nil)))
		}
	}
}

// guessCastURLType guesses the type of URL. This approach all allows prepareURL
// functions to accurately report failure causes.
func guessCastURLType(sourceURL string, targetURLs []string) castURLsType {
	if targetURLs == nil || len(targetURLs) == 0 { // Target is empty
		return castURLsTypeInvalid
	}
	if sourceURL == "" { // Source is empty
		return castURLsTypeInvalid
	}
	for _, targetURL := range targetURLs {
		if targetURL == "" { // One of the target is empty
			return castURLsTypeInvalid
		}
	}

	if isURLRecursive(sourceURL) { // Type C
		return castURLsTypeC
	} // else Type A or Type B
	for _, targetURL := range targetURLs {
		if isTargetURLDir(targetURL) { // Type B
			return castURLsTypeB
		}
	} // else Type A
	return castURLsTypeA
}

// prepareSingleCastURLTypeA - prepares a single source and single target argument for casting.
func prepareSingleCastURLsTypeA(sourceURL string, targetURL string) castURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil { // Source does not exist or insufficient privileges.
		return castURLs{Error: NewIodine(iodine.New(err, nil))}
	}
	if !sourceContent.Type.IsRegular() { // Source is not a regular file
		return castURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	// All OK.. We can proceed. Type A
	sourceContent.Name = sourceURL
	return castURLs{SourceContent: sourceContent, TargetContents: []*client.Content{{Name: targetURL}}}
}

// prepareCastURLsTypeA - A: cast(f, f) -> cast(f, f)
func prepareCastURLsTypeA(sourceURL string, targetURLs []string) castURLs {
	var sURLs castURLs
	for _, targetURL := range targetURLs { // Prepare each target separately
		URLs := prepareSingleCastURLsTypeA(sourceURL, targetURL)
		if URLs.Error != nil {
			return castURLs{Error: NewIodine(iodine.New(URLs.Error, nil))}
		}
		sURLs.SourceContent = URLs.SourceContent
		sURLs.TargetContents = append(sURLs.TargetContents, URLs.TargetContents...)
	}
	return sURLs
}

// prepareSingleCastURLsTypeB - prepares a single target and single source URLs for casting.
func prepareSingleCastURLsTypeB(sourceURL string, targetURL string) castURLs {
	_, sourceContent, err := url2Stat(sourceURL)
	if err != nil {
		// Source does not exist or insufficient privileges.
		return castURLs{Error: NewIodine(iodine.New(err, nil))}
	}

	if !sourceContent.Type.IsRegular() {
		// Source is not a regular file.
		return castURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	_, targetContent, err := url2Stat(targetURL)
	if err != nil {
		// Source and target are files. Already reduced to Type A.
		return prepareSingleCastURLsTypeA(sourceURL, targetURL)
	}

	if targetContent.Type.IsRegular() { // File to File
		// Source and target are files. Already reduced to Type A.
		return prepareSingleCastURLsTypeA(sourceURL, targetURL)
	}

	// Source is a file, target is a folder and exists.
	sourceURLParse, err := client.Parse(sourceURL)
	if err != nil {
		return castURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
	}

	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return castURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
	}
	// Reduce Type B to Type A.
	targetURLParse.Path = filepath.Join(targetURLParse.Path, filepath.Base(sourceURLParse.Path))
	return prepareSingleCastURLsTypeA(sourceURL, targetURLParse.String())
}

// prepareCastURLsTypeB - B: cast(f, d) -> cast(f, d/f) -> A
func prepareCastURLsTypeB(sourceURL string, targetURLs []string) castURLs {
	var sURLs castURLs
	for _, targetURL := range targetURLs {
		URLs := prepareSingleCastURLsTypeB(sourceURL, targetURL)
		if URLs.Error != nil {
			return castURLs{Error: NewIodine(iodine.New(URLs.Error, nil))}
		}
		sURLs.SourceContent = URLs.SourceContent
		sURLs.TargetContents = append(sURLs.TargetContents, URLs.TargetContents[0])
	}
	return sURLs
}

// prepareCastURLsTypeC - C:
func prepareCastURLsTypeC(sourceURL string, targetURLs []string) <-chan castURLs {
	castURLsCh := make(chan castURLs)
	go func() {
		defer close(castURLsCh)
		if !isURLRecursive(sourceURL) {
			// Source is not of recursive type.
			castURLsCh <- castURLs{Error: NewIodine(iodine.New(errSourceNotRecursive{URL: sourceURL}, nil))}
			return
		}
		// add `/` after trimming off `...` to emulate folders
		sourceURL = stripRecursiveURL(sourceURL)
		sourceClient, sourceContent, err := url2Stat(sourceURL)
		// Source exist?
		if err != nil {
			// Source does not exist or insufficient privileges.
			castURLsCh <- castURLs{Error: NewIodine(iodine.New(err, nil))}
			return
		}

		if !sourceContent.Type.IsDir() {
			// Source is not a dir.
			castURLsCh <- castURLs{Error: NewIodine(iodine.New(errSourceIsNotDir{URL: sourceURL}, nil))}
			return
		}

		for sourceContent := range sourceClient.List(true) {
			if sourceContent.Err != nil {
				// Listing failed.
				castURLsCh <- castURLs{Error: NewIodine(iodine.New(sourceContent.Err, nil))}
				continue
			}
			if !sourceContent.Content.Type.IsRegular() {
				// Source is not a regular file. Skip it for cast.
				continue
			}
			// All OK.. We can proceed. Type B: source is a file, target is a folder and exists.
			sourceURLParse, err := client.Parse(sourceURL)
			if err != nil {
				castURLsCh <- castURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, nil))}
				continue
			}
			var newTargetURLs []string
			var sourceContentParse *client.URL
			for _, targetURL := range targetURLs {
				targetURLParse, err := client.Parse(targetURL)
				if err != nil {
					castURLsCh <- castURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
					continue
				}
				sourceURLDelimited := sourceURLParse.String()[:strings.LastIndex(sourceURLParse.String(),
					string(sourceURLParse.Separator))+1]
				sourceContentName := sourceContent.Content.Name
				sourceContentURL := sourceURLDelimited + sourceContentName
				sourceContentParse, err = client.Parse(sourceContentURL)
				if err != nil {
					castURLsCh <- castURLs{Error: NewIodine(iodine.New(errInvalidSource{URL: sourceContentName}, nil))}
					continue
				}
				// Construct target path from recursive path of source without its prefix dir.
				newTargetURLParse := *targetURLParse
				newTargetURLParse.Path = filepath.Join(newTargetURLParse.Path, sourceContentName)
				newTargetURLs = append(newTargetURLs, newTargetURLParse.String())
			}
			castURLsCh <- prepareCastURLsTypeA(sourceContentParse.String(), newTargetURLs)
		}
	}()
	return castURLsCh
}

// prepareCastURLs - prepares target and source URLs for casting.
func prepareCastURLs(sourceURL string, targetURLs []string) <-chan castURLs {
	castURLsCh := make(chan castURLs)
	go func() {
		defer close(castURLsCh)
		switch guessCastURLType(sourceURL, targetURLs) {
		case castURLsTypeA:
			castURLsCh <- prepareCastURLsTypeA(sourceURL, targetURLs)
		case castURLsTypeB:
			castURLsCh <- prepareCastURLsTypeB(sourceURL, targetURLs)
		case castURLsTypeC:
			for sURLs := range prepareCastURLsTypeC(sourceURL, targetURLs) {
				castURLsCh <- sURLs
			}
		default:
			castURLsCh <- castURLs{Error: NewIodine(iodine.New(errInvalidArgument{}, nil))}
		}
	}()
	return castURLsCh
}
