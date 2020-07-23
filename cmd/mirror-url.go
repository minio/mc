/*
 * MinIO Client (C) 2015, 2016 MinIO, Inc.
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
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/wildcard"
)

//
//   * MIRROR ARGS - VALID CASES
//   =========================
//   mirror(d1..., d2) -> []mirror(d1/f, d2/d1/f)

// checkMirrorSyntax(URLs []string)
func checkMirrorSyntax(ctx context.Context, cliCtx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	if len(cliCtx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(cliCtx, "mirror", 1) // last argument is exit code.
	}

	// extract URLs.
	URLs := cliCtx.Args()
	srcURL := URLs[0]
	tgtURL := URLs[1]

	if cliCtx.Bool("force") && cliCtx.Bool("remove") {
		errorIf(errInvalidArgument().Trace(URLs...), "`--force` is deprecated, please use `--overwrite` instead with `--remove` for the same functionality.")
	} else if cliCtx.Bool("force") {
		errorIf(errInvalidArgument().Trace(URLs...), "`--force` is deprecated, please use `--overwrite` instead for the same functionality.")
	}

	tgtClientURL := newClientURL(tgtURL)
	if tgtClientURL.Host != "" {
		if tgtClientURL.Path == string(tgtClientURL.Separator) {
			fatalIf(errInvalidArgument().Trace(tgtURL),
				fmt.Sprintf("Target `%s` does not contain bucket name.", tgtURL))
		}
	}

	_, expandedSourcePath, _ := mustExpandAlias(srcURL)
	srcClient := newClientURL(expandedSourcePath)
	_, expandedTargetPath, _ := mustExpandAlias(tgtURL)
	destClient := newClientURL(expandedTargetPath)

	// Mirror with preserve option on windows
	// only works for object storage to object storage
	if runtime.GOOS == "windows" && cliCtx.Bool("a") {
		if srcClient.Type == fileSystem || destClient.Type == fileSystem {
			errorIf(errInvalidArgument(), "Preserve functionality on windows support object storage to object storage transfer only.")
		}
	}

	/****** Generic rules *******/
	if !cliCtx.Bool("watch") && !cliCtx.Bool("active-active") && !cliCtx.Bool("multi-master") {
		_, srcContent, err := url2Stat(ctx, srcURL, "", false, encKeyDB, time.Time{})
		// incomplete uploads are not necessary for mirror operation, no need to verify for them.
		isIncomplete := false
		if err != nil && !isURLPrefixExists(srcURL, isIncomplete) {
			fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")
		}

		if err == nil {
			if !srcContent.Type.IsDir() {
				fatalIf(errInvalidArgument().Trace(srcContent.URL.String(), srcContent.Type.String()), fmt.Sprintf("Source `%s` is not a folder. Only folders are supported by mirror command.", srcURL))
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

func deltaSourceTarget(ctx context.Context, sourceURL, targetURL string, opts mirrorOptions, URLsCh chan<- URLs) {
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
	for diffMsg := range objectDifference(ctx, sourceClnt, targetClnt, sourceURL, targetURL, opts.isMetadata) {
		if diffMsg.Error != nil {
			// Send all errors through the channel
			URLsCh <- URLs{Error: diffMsg.Error}
			continue
		}

		srcSuffix := strings.TrimPrefix(diffMsg.FirstURL, sourceURL)
		//Skip the source object if it matches the Exclude options provided
		if matchExcludeOptions(opts.excludeOptions, srcSuffix) {
			continue
		}

		tgtSuffix := strings.TrimPrefix(diffMsg.SecondURL, targetURL)
		//Skip the target object if it matches the Exclude options provided
		if matchExcludeOptions(opts.excludeOptions, tgtSuffix) {
			continue
		}

		switch diffMsg.Diff {
		case differInNone:
			// No difference, continue.
		case differInType:
			URLsCh <- URLs{Error: errInvalidTarget(diffMsg.SecondURL)}
		case differInSize, differInMetadata, differInAASourceMTime:
			if !opts.isOverwrite && !opts.isFake && !opts.activeActive {
				// Size or time or etag differs but --overwrite not set.
				URLsCh <- URLs{Error: errOverWriteNotAllowed(diffMsg.SecondURL)}
				continue
			}

			sourceSuffix := strings.TrimPrefix(diffMsg.FirstURL, sourceURL)
			// Either available only in source or size differs and force is set
			targetPath := urlJoinPath(targetURL, sourceSuffix)
			sourceContent := diffMsg.firstContent
			targetContent := &ClientContent{URL: *newClientURL(targetPath)}
			URLsCh <- URLs{
				SourceAlias:   sourceAlias,
				SourceContent: sourceContent,
				TargetAlias:   targetAlias,
				TargetContent: targetContent,
			}
		case differInFirst:
			// Only in first, always copy.
			sourceSuffix := strings.TrimPrefix(diffMsg.FirstURL, sourceURL)
			targetPath := urlJoinPath(targetURL, sourceSuffix)
			sourceContent := diffMsg.firstContent
			targetContent := &ClientContent{URL: *newClientURL(targetPath)}
			URLsCh <- URLs{
				SourceAlias:   sourceAlias,
				SourceContent: sourceContent,
				TargetAlias:   targetAlias,
				TargetContent: targetContent,
			}
		case differInSecond:
			if !opts.isRemove && !opts.isFake {
				continue
			}
			URLsCh <- URLs{
				TargetAlias:   targetAlias,
				TargetContent: diffMsg.secondContent,
			}
		default:
			URLsCh <- URLs{
				Error: errUnrecognizedDiffType(diffMsg.Diff).Trace(diffMsg.FirstURL, diffMsg.SecondURL),
			}
		}
	}
}

type mirrorOptions struct {
	isFake, isOverwrite, activeActive bool
	isWatch, isRemove, isMetadata     bool
	excludeOptions                    []string
	encKeyDB                          map[string][]prefixSSEPair
	md5, disableMultipart             bool
	olderThan, newerThan              string
	storageClass                      string
	userMetadata                      map[string]string
}

// Prepares urls that need to be copied or removed based on requested options.
func prepareMirrorURLs(ctx context.Context, sourceURL string, targetURL string, opts mirrorOptions) <-chan URLs {
	URLsCh := make(chan URLs)
	go deltaSourceTarget(ctx, sourceURL, targetURL, opts, URLsCh)
	return URLsCh
}
