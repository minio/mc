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
	"os"
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
	castURLsTypeD
)

//   NOTE: All the parse rules should reduced to A: Cast(Source, []Target).
//
//   * CAST ARGS - VALID CASES
//   =========================
//   A: cast(f, []f) -> cast(f, []f)
//   B: cast(f, [](d | f)) -> cast(f, [](d/f | f)) -> A:
//   C: cast(d1..., [](d2)) -> []cast(d1/f, [](d1/d2/f)) -> []A:

// checkCastSyntax(URLs []string)
func checkCastSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cast", 1) // last argument is exit code.
	}

	// extract URLs.
	URLs, err := args2URLs(ctx.Args())
	if err != nil {
		console.Fatalf("One or more unknown URL types found %s. %s\n", ctx.Args(), err)
	}

	srcURL := URLs[0]
	tgtURLs := URLs[1:]

	if !isURLRecursive(srcURL) {
		_, srcContent, err := url2Stat(srcURL)
		// Source exist?.
		if err != nil {
			console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, iodine.New(err, nil))
		}

		// Rule A & B validation.
		if srcContent.Type.IsRegular() {
			// All targets should be a valid file or folder.
			for _, tgtURL := range tgtURLs {
				_, tgtContent, err := url2Stat(tgtURL)
				// Target exist?
				if err != nil {
					console.Fatalf("Unable to stat target ‘%s’. %s\n", tgtURL, iodine.New(err, nil))
				}

				if !tgtContent.Type.IsRegular() {
					if !tgtContent.Type.IsDir() {
						console.Fatalf("Target ‘%s’ is not a valid file or directory.\n", tgtURL)
					}
				}
			}
		} else { // Source is not a file.
			if srcContent.Type.IsDir() {
				console.Fatalf("Source ‘%s’ is a directory. Please use ‘%s...’ ellipses sufix to copy a directory and its contents recursively.\n", srcURL, srcURL)
			}
			console.Fatalf("Source ‘%s’ is not a regular file.\n", srcURL)
		}

	} else { // Rule C validation.
		srcURL = stripRecursiveURL(srcURL)
		_, srcContent, err := url2Stat(srcURL)
		// Source exist?.
		if err != nil {
			console.Fatalf("Unable to stat source ‘%s’. %s\n", srcURL, iodine.New(err, nil))
		}

		if srcContent.Type.IsRegular() { // Elipses is supported only for directories.
			console.Fatalf("Source ‘%s’ is not a directory. %s\n", stripRecursiveURL(srcURL), iodine.New(err, nil))
		}

		// All targets should be a valid file or folder.
		for _, tgtURL := range tgtURLs {
			_, tgtContent, err := url2Stat(tgtURL)
			// Target exist?
			if err != nil {
				console.Fatalf("Unable to stat target directory ‘%s’. %s\n", tgtURL, iodine.New(err, nil))
			}
			if !tgtContent.Type.IsDir() {
				console.Fatalf("Target ‘%s’ is not a directory.\n", tgtURL)
			}
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
	targetClient, err := target2Client(targetURL)
	if err != nil {
		return castURLs{Error: NewIodine(iodine.New(err, nil))}
	}
	// Target exists?
	targetContent, err := targetClient.Stat()
	if err == nil { // Target exists.
		if !targetContent.Type.IsRegular() { // Target is not a regular file
			return castURLs{Error: NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, nil))}
		}
		var targetContents []*client.Content
		targetContents = append(targetContents, targetContent)
		return castURLs{SourceContent: sourceContent, TargetContents: targetContents}
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
	if os.IsNotExist(iodine.ToError(err)) {
		// Source and target are files. Already reduced to Type A.
		return prepareSingleCastURLsTypeA(sourceURL, targetURL)
	}
	if err != nil {
		return castURLs{Error: NewIodine(iodine.New(err, nil))}
	}

	if targetContent.Type.IsRegular() { // File to File
		// Source and target are files. Already reduced to Type A.
		return prepareSingleCastURLsTypeA(sourceURL, targetURL)
	}

	// Source is a file, target is a directory and exists.
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
		// add `/` after trimming off `...` to emulate directories
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

		// Type C requires all targets to be a dir and it should exist.
		for _, targetURL := range targetURLs {
			_, targetContent, err := url2Stat(targetURL)
			// Target exist?
			if err != nil {
				// Target does not exist.
				castURLsCh <- castURLs{Error: NewIodine(iodine.New(errTargetNotFound{URL: targetURL}, nil))}
				return
			}

			if !targetContent.Type.IsDir() {
				// Target exists, but is not a directory.
				castURLsCh <- castURLs{Error: NewIodine(iodine.New(errTargetIsNotDir{URL: targetURL}, nil))}
				return
			}
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
			// All OK.. We can proceed. Type B: source is a file, target is a directory and exists.
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
		case castURLsType(castURLsTypeA):
			castURLsCh <- prepareCastURLsTypeA(sourceURL, targetURLs)
		case castURLsType(castURLsTypeB):
			castURLsCh <- prepareCastURLsTypeB(sourceURL, targetURLs)
		case castURLsType(castURLsTypeC):
			for sURLs := range prepareCastURLsTypeC(sourceURL, targetURLs) {
				castURLsCh <- sURLs
			}
		default:
			castURLsCh <- castURLs{Error: NewIodine(iodine.New(errInvalidArgument{}, nil))}
		}
	}()
	return castURLsCh
}
