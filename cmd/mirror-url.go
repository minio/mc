/*
 * Minio Client (C) 2015, 2016 Minio, Inc.
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

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/wildcard"
)

//
//   * MIRROR ARGS - VALID CASES
//   =========================
//   mirror(d1..., d2) -> []mirror(d1/f, d2/d1/f)

// checkMirrorSyntax(URLs []string)
func checkMirrorSyntax(ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "mirror", 1) // last argument is exit code.
	}

	// extract URLs.
	URLs := ctx.Args()
	srcURL := URLs[0]
	tgtURL := URLs[1]

	if ctx.Bool("force") && ctx.Bool("remove") {
		errorIf(errInvalidArgument().Trace(URLs...), "`--force` is deprecated please use `--overwrite` instead with `--remove` for the same functionality.")
	} else if ctx.Bool("force") {
		errorIf(errInvalidArgument().Trace(URLs...), "`--force` is deprecated please use `--overwrite` instead for the same functionality.")
	}

	tgtClientURL := newClientURL(tgtURL)
	if tgtClientURL.Host != "" {
		if tgtClientURL.Path == string(tgtClientURL.Separator) {
			fatalIf(errInvalidArgument().Trace(tgtURL),
				fmt.Sprintf("Target `%s` does not contain bucket name.", tgtURL))
		}
	}

	/****** Generic rules *******/
	if !ctx.Bool("watch") {
		c, srcContent, err := url2Stat(srcURL, false, encKeyDB)
		// incomplete uploads are not necessary for copy operation, no need to verify for them.
		isIncomplete := false
		if err != nil && !isURLPrefixExists(srcURL, isIncomplete) {
			errorIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")
		}

		if err == nil {
			if !srcContent.Type.IsDir() {
				fatalIf(errInvalidArgument().Trace(srcContent.URL.String(), srcContent.Type.String()), fmt.Sprintf("Source `%s` is not a folder. Only folders are supported by mirror command.", srcURL))
			}

			// Disallow mirroring a directory to itself
			if isURLContains(srcURL, tgtURL, string(c.GetURL().Separator)) {
				fatalIf(errInvalidArgument().Trace(), "Mirroring a folder into itself is not allowed.")
			}
		}
	}

}

func matchExcludeOptions(excludeOptions []string, srcSuffix string) bool {
	for _, pattern := range excludeOptions {
		if wildcard.Match(pattern, srcSuffix) {
			return true
		}
	}
	return false
}

func deltaSourceTarget(sourceURL, targetURL string, isFake, isOverwrite, isRemove bool, excludeOptions []string, URLsCh chan<- URLs, encKeyDB map[string][]prefixSSEPair) {
	// source and targets are always directories
	sourceSeparator := string(newClientURL(sourceURL).Separator)
	if !strings.HasSuffix(sourceURL, sourceSeparator) {
		sourceURL = sourceURL + sourceSeparator
	}
	targetSeparator := string(newClientURL(targetURL).Separator)
	if !strings.HasSuffix(targetURL, targetSeparator) {
		targetURL = targetURL + targetSeparator
	}

	// Extract alias and expanded URL
	sourceAlias, sourceURL, _ := mustExpandAlias(sourceURL)
	targetAlias, targetURL, _ := mustExpandAlias(targetURL)

	defer close(URLsCh)

	sourceClnt, err := newClientFromAlias(sourceAlias, sourceURL)
	if err != nil {
		URLsCh <- URLs{Error: err.Trace(sourceAlias, sourceURL)}
		return
	}

	targetClnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		URLsCh <- URLs{Error: err.Trace(targetAlias, targetURL)}
		return
	}

	// List both source and target, compare and return values through channel.
	for diffMsg := range objectDifference(sourceClnt, targetClnt, sourceURL, targetURL) {
		if diffMsg.Error != nil {
			// Send all errors through the channel
			URLsCh <- URLs{Error: diffMsg.Error}
			return
		}

		srcSuffix := strings.TrimPrefix(diffMsg.FirstURL, sourceURL)
		//Skip the source object if it matches the Exclude options provided
		if matchExcludeOptions(excludeOptions, srcSuffix) {
			continue
		}

		tgtSuffix := strings.TrimPrefix(diffMsg.SecondURL, targetURL)
		//Skip the target object if it matches the Exclude options provided
		if matchExcludeOptions(excludeOptions, tgtSuffix) {
			continue
		}

		sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceClnt.GetURL().Path))
		srcSSEKey := getSSEKey(sourcePath, encKeyDB[sourceAlias])
		targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetClnt.GetURL().Path))
		tgtSSEKey := getSSEKey(targetPath, encKeyDB[targetAlias])

		switch diffMsg.Diff {
		case differInNone:
			// No difference, continue.
		case differInType:
			URLsCh <- URLs{Error: errInvalidTarget(diffMsg.SecondURL)}
		case differInSize, differInTime:
			if !isOverwrite && !isFake {
				// Size or time differs but --overwrite not set.
				URLsCh <- URLs{Error: errOverWriteNotAllowed(diffMsg.SecondURL)}
				continue
			}

			sourceSuffix := strings.TrimPrefix(diffMsg.FirstURL, sourceURL)
			// Either available only in source or size differs and force is set
			targetPath := urlJoinPath(targetURL, sourceSuffix)
			sourceContent := diffMsg.firstContent
			targetContent := &clientContent{URL: *newClientURL(targetPath)}
			URLsCh <- URLs{
				SourceAlias:   sourceAlias,
				SourceContent: sourceContent,
				TargetAlias:   targetAlias,
				TargetContent: targetContent,
				SrcSSEKey:     srcSSEKey,
				TgtSSEKey:     tgtSSEKey,
			}
		case differInFirst:
			// Only in first, always copy.
			sourceSuffix := strings.TrimPrefix(diffMsg.FirstURL, sourceURL)
			targetPath := urlJoinPath(targetURL, sourceSuffix)
			sourceContent := diffMsg.firstContent
			targetContent := &clientContent{URL: *newClientURL(targetPath)}
			URLsCh <- URLs{
				SourceAlias:   sourceAlias,
				SourceContent: sourceContent,
				TargetAlias:   targetAlias,
				TargetContent: targetContent,
				SrcSSEKey:     srcSSEKey,
				TgtSSEKey:     tgtSSEKey,
			}
		case differInSecond:
			if !isRemove && !isFake {
				continue
			}
			URLsCh <- URLs{
				TargetAlias:   targetAlias,
				TargetContent: diffMsg.secondContent,
				TgtSSEKey:     tgtSSEKey,
			}
		default:
			URLsCh <- URLs{
				Error: errUnrecognizedDiffType(diffMsg.Diff).Trace(diffMsg.FirstURL, diffMsg.SecondURL),
			}
		}
	}
}

// Prepares urls that need to be copied or removed based on requested options.
func prepareMirrorURLs(sourceURL string, targetURL string, isFake, isOverwrite, isRemove bool, excludeOptions []string, encKeyDB map[string][]prefixSSEPair) <-chan URLs {
	URLsCh := make(chan URLs)
	go deltaSourceTarget(sourceURL, targetURL, isFake, isOverwrite, isRemove, excludeOptions, URLsCh, encKeyDB)
	return URLsCh
}
