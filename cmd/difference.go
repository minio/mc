// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	// golang does not support flat keys for path matching, find does

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"golang.org/x/text/unicode/norm"
)

// differType difference in type.
type differType int

const (
	differInUnknown       differType = iota
	differInNone                     // does not differ
	differInSize                     // differs in size
	differInMetadata                 // differs in metadata
	differInType                     // differs in type, exfile/directory
	differInFirst                    // only in source (FIRST)
	differInSecond                   // only in target (SECOND)
	differInAASourceMTime            // differs in active-active source modtime
)

func (d differType) String() string {
	switch d {
	case differInNone:
		return ""
	case differInSize:
		return "size"
	case differInMetadata:
		return "metadata"
	case differInAASourceMTime:
		return "mm-source-mtime"
	case differInType:
		return "type"
	case differInFirst:
		return "only-in-first"
	case differInSecond:
		return "only-in-second"
	}
	return "unknown"
}

const activeActiveSourceModTimeKey = "X-Amz-Meta-Mm-Source-Mtime"

func getSourceModTimeKey(metadata map[string]string) string {
	if metadata[activeActiveSourceModTimeKey] != "" {
		return metadata[activeActiveSourceModTimeKey]
	}
	if metadata[strings.ToLower(activeActiveSourceModTimeKey)] != "" {
		return metadata[strings.ToLower(activeActiveSourceModTimeKey)]
	}
	if metadata[strings.ToLower("Mm-Source-Mtime")] != "" {
		return metadata[strings.ToLower("Mm-Source-Mtime")]
	}
	if metadata["Mm-Source-Mtime"] != "" {
		return metadata["Mm-Source-Mtime"]
	}
	return ""
}

// layerDifference performs a breadth-first search (BFS) comparison between source and target.
// Unlike the standard recursive listing approach, this function traverses the object hierarchy
// layer by layer (directory by directory), which prevents overwhelming the server with
// large recursive listing operations that could cause timeouts or connection failures.
//
// This approach is especially useful for buckets containing millions of objects where
// a standard recursive listing might cause server-side resource exhaustion. By exploring
// the hierarchy level by level and comparing objects at each layer, this function provides
// a more scalable solution for large object stores.
//
// The BFS approach:
// 1. Starts with the root prefix ("") for both source and target
// 2. Lists objects at the current level/prefix (non-recursively)
// 3. Compares objects found at this level
// 4. Queues any directories found for exploration in the next iteration
// 5. Continues until all directories in both source and target are explored
//
// This is enabled with the --bfs parameter to avoid the limitations of recursive listing.
func layerDifference(ctx context.Context, sourceClnt, targetClnt Client, opts mirrorOptions) chan diffMessage {
	diffCh := make(chan diffMessage, 10000)

	go func() {
		defer close(diffCh)

		// Channels to feed items found by BFS into the difference engine
		srcClientCh := make(chan *ClientContent, 1000)
		tgtClientCh := make(chan *ClientContent, 1000)

		// Goroutine to perform BFS on the source
		go func() {
			defer close(srcClientCh)
			// Queue for *relative object prefixes* to explore
			queue := []string{""} // "" represents the root prefix

			for len(queue) > 0 {
				// Dequeue the next relative prefix
				prefix := queue[0]
				queue = queue[1:]

				// List items at the current prefix level using the relative prefix
				listCtx, listCancel := context.WithCancel(ctx)
				contentsCh := sourceClnt.List(listCtx, ListOptions{
					Recursive:    false, // List only the current level
					WithMetadata: opts.isMetadata,
					ShowDir:      DirLast, // Ensure directories are listed
					Prefix:       prefix,  // Pass the relative prefix
				})

				for content := range contentsCh {
					select {
					case <-ctx.Done():
						listCancel()
						return
					default:
						if content != nil && content.Err != nil {
							srcClientCh <- content
							listCancel()
							continue
						}
						if content == nil {
							continue
						}

						// Send the valid content (file or directory) for comparison
						srcClientCh <- content

						// If it's a directory, queue its *relative object key* for the next level
						if content.Type.IsDir() {
							relativeKey := content.ObjectKey // Get the relative key
							// Prevent infinite loops: don't re-queue the prefix we just listed,
							// especially the root ("") which might list itself as "/" depending on backend.
							// Also check if ObjectKey is populated.
							if relativeKey != "" && relativeKey != prefix {
								// Ensure the key ends with a separator if it's a directory prefix
								// The S3 ListObjects usually returns directory keys ending with '/'
								if !strings.HasSuffix(relativeKey, string(content.URL.Separator)) {
									// This case might indicate a non-standard directory representation, handle cautiously
									// For standard S3, common prefixes already end in '/'
									// If needed, append separator: relativeKey += string(content.URL.Separator)
								}
								// Add the relative key (prefix) to the queue
								queue = append(queue, relativeKey)
							}
						}
					}
				}
				listCancel()
			}
		}()

		// Goroutine to perform BFS on the target (symmetric to the source)
		go func() {
			defer close(tgtClientCh)
			// Queue for *relative object prefixes*
			queue := []string{""}

			for len(queue) > 0 {
				prefix := queue[0]
				queue = queue[1:]

				listCtx, listCancel := context.WithCancel(ctx)
				contentsCh := targetClnt.List(listCtx, ListOptions{
					Recursive:    false,
					WithMetadata: opts.isMetadata,
					ShowDir:      DirLast,
					Prefix:       prefix, // Pass the relative prefix
				})

				for content := range contentsCh {
					select {
					case <-ctx.Done():
						listCancel()
						return
					default:
						if content != nil && content.Err != nil {
							tgtClientCh <- content
							listCancel()
							continue
						}
						if content == nil {
							continue
						}

						tgtClientCh <- content

						// If it's a directory, queue its *relative object key*
						if content.Type.IsDir() {
							relativeKey := content.ObjectKey
							if relativeKey != "" && relativeKey != prefix {
								// Ensure trailing slash if needed (usually present from S3 List)
								if !strings.HasSuffix(relativeKey, string(content.URL.Separator)) {
									// Handle non-standard directory representation if necessary
								}
								queue = append(queue, relativeKey)
							}
						}
					}
				}
				listCancel()
			}
		}()

		// Comparison logic remains the same
		err := differenceInternal(
			sourceClnt.GetURL().String(),
			srcClientCh,
			targetClnt.GetURL().String(),
			tgtClientCh,
			opts,
			false, // returnSimilar is false
			diffCh,
		)

		if err != nil {
			select {
			case <-ctx.Done():
			default:
				diffCh <- diffMessage{Error: err}
			}
		}
	}()

	return diffCh
}

// activeActiveModTimeUpdated tries to calculate if the object copy in the target
// is older than the one in the source by comparing the modtime of the data.
func activeActiveModTimeUpdated(src, dst *ClientContent) bool {
	if src == nil || dst == nil {
		return false
	}

	if src.Time.IsZero() || dst.Time.IsZero() {
		// This should only happen in a messy environment
		// but we are returning false anyway so the caller
		// function won't take any action.
		return false
	}

	srcActualModTime := src.Time
	dstActualModTime := dst.Time

	srcModTime := getSourceModTimeKey(src.UserMetadata)
	dstModTime := getSourceModTimeKey(dst.UserMetadata)
	if srcModTime == "" && dstModTime == "" {
		// No active-active mirror context found, fallback to modTimes presented
		// by the client content
		return srcActualModTime.After(dstActualModTime)
	}

	var srcOriginLastModified, dstOriginLastModified time.Time
	var err error
	if srcModTime != "" {
		srcOriginLastModified, err = time.Parse(time.RFC3339Nano, srcModTime)
		if err != nil {
			// failure to parse source modTime, modTime tampered ignore the file
			return false
		}
	}
	if dstModTime != "" {
		dstOriginLastModified, err = time.Parse(time.RFC3339Nano, dstModTime)
		if err != nil {
			// failure to parse source modTime, modTime tampered ignore the file
			return false
		}
	}

	if !srcOriginLastModified.IsZero() && srcOriginLastModified.After(src.Time) {
		srcActualModTime = srcOriginLastModified
	}

	if !dstOriginLastModified.IsZero() && dstOriginLastModified.After(dst.Time) {
		dstActualModTime = dstOriginLastModified
	}

	return srcActualModTime.After(dstActualModTime)
}

func metadataEqual(m1, m2 map[string]string) bool {
	for k, v := range m1 {
		if k == activeActiveSourceModTimeKey {
			continue
		}
		if k == strings.ToLower(activeActiveSourceModTimeKey) {
			continue
		}
		if m2[k] != v {
			return false
		}
	}
	for k, v := range m2 {
		if k == activeActiveSourceModTimeKey {
			continue
		}
		if k == strings.ToLower(activeActiveSourceModTimeKey) {
			continue
		}
		if m1[k] != v {
			return false
		}
	}
	return true
}

func bucketObjectDifference(ctx context.Context, sourceClnt, targetClnt Client) (diffCh chan diffMessage) {
	return objectDifference(ctx, sourceClnt, targetClnt, mirrorOptions{
		isMetadata: false,
	})
}

func objectDifference(ctx context.Context, sourceClnt, targetClnt Client, opts mirrorOptions) chan diffMessage {
	if opts.bfs {
		// Use layer-by-layer difference for regular objects
		return layerDifference(ctx, sourceClnt, targetClnt, opts)
	}

	sourceURL := sourceClnt.GetURL().String()
	sourceCh := sourceClnt.List(ctx, ListOptions{Recursive: true, WithMetadata: opts.isMetadata, ShowDir: DirNone})

	targetURL := targetClnt.GetURL().String()
	targetCh := targetClnt.List(ctx, ListOptions{Recursive: true, WithMetadata: opts.isMetadata, ShowDir: DirNone})

	return difference(sourceURL, sourceCh, targetURL, targetCh, opts, false)
}

func bucketDifference(ctx context.Context, sourceClnt, targetClnt Client, opts mirrorOptions) (diffCh chan diffMessage) {
	sourceURL := sourceClnt.GetURL().String()
	sourceCh := make(chan *ClientContent)

	go func() {
		defer close(sourceCh)
		buckets, err := sourceClnt.ListBuckets(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
			case sourceCh <- &ClientContent{Err: err}:
			}
			return
		}
		for _, b := range buckets {
			select {
			case <-ctx.Done():
				return
			case sourceCh <- b:
			}
		}
	}()

	targetURL := targetClnt.GetURL().String()
	targetCh := make(chan *ClientContent)
	go func() {
		defer close(targetCh)
		buckets, err := targetClnt.ListBuckets(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
			case targetCh <- &ClientContent{Err: err}:
			}
			return
		}
		for _, b := range buckets {
			select {
			case <-ctx.Done():
				return
			case targetCh <- b:
			}
		}
	}()

	return difference(sourceURL, sourceCh, targetURL, targetCh, opts, false)
}

func differenceInternal(sourceURL string,
	srcCh <-chan *ClientContent,
	targetURL string,
	tgtCh <-chan *ClientContent,
	opts mirrorOptions,
	returnSimilar bool,
	diffCh chan<- diffMessage,
) *probe.Error {
	// Pop first entries from the source and targets
	srcCtnt, srcOk := <-srcCh
	tgtCtnt, tgtOk := <-tgtCh

	var srcEOF, tgtEOF bool

	for {
		srcEOF = !srcOk
		tgtEOF = !tgtOk

		// No objects from source AND target: Finish
		if opts.sourceListingOnly {
			if srcEOF {
				break
			}
		} else {
			if srcEOF && tgtEOF {
				break
			}
		}

		if !srcEOF && srcCtnt.Err != nil {
			return srcCtnt.Err.Trace(sourceURL, targetURL)
		}

		if !tgtEOF && tgtCtnt.Err != nil {
			return tgtCtnt.Err.Trace(sourceURL, targetURL)
		}

		// If source doesn't have objects anymore, comparison becomes obvious
		if srcEOF {
			diffCh <- diffMessage{
				SecondURL:     tgtCtnt.URL.String(),
				Diff:          differInSecond,
				secondContent: tgtCtnt,
			}
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}

		// The same for target
		if tgtEOF {
			diffCh <- diffMessage{
				FirstURL:     srcCtnt.URL.String(),
				Diff:         differInFirst,
				firstContent: srcCtnt,
			}
			srcCtnt, srcOk = <-srcCh
			continue
		}

		srcSuffix := strings.TrimPrefix(srcCtnt.URL.String(), sourceURL)
		tgtSuffix := strings.TrimPrefix(tgtCtnt.URL.String(), targetURL)

		current := urlJoinPath(targetURL, srcSuffix)
		expected := urlJoinPath(targetURL, tgtSuffix)

		if !utf8.ValidString(srcSuffix) {
			// Error. Keys must be valid UTF-8.
			diffCh <- diffMessage{Error: errInvalidSource(current).Trace()}
			srcCtnt, srcOk = <-srcCh
			continue
		}
		if !utf8.ValidString(tgtSuffix) {
			// Error. Keys must be valid UTF-8.
			diffCh <- diffMessage{Error: errInvalidTarget(expected).Trace()}
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}

		// Normalize to avoid situations where multiple byte representations are possible.
		// e.g. 'ä' can be represented as precomposed U+00E4 (UTF-8 0xc3a4) or decomposed
		// U+0061 U+0308 (UTF-8 0x61cc88).
		normalizedCurrent := norm.NFC.String(current)
		normalizedExpected := norm.NFC.String(expected)

		if normalizedExpected > normalizedCurrent {
			diffCh <- diffMessage{
				FirstURL:     srcCtnt.URL.String(),
				Diff:         differInFirst,
				firstContent: srcCtnt,
			}
			srcCtnt, srcOk = <-srcCh
			continue
		}
		if normalizedExpected == normalizedCurrent {
			srcType, tgtType := srcCtnt.Type, tgtCtnt.Type
			srcSize, tgtSize := srcCtnt.Size, tgtCtnt.Size
			if srcType.IsRegular() && !tgtType.IsRegular() ||
				!srcType.IsRegular() && tgtType.IsRegular() {
				// Type differs. Source is never a directory.
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInType,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
				continue
			}
			if srcSize != tgtSize {
				// Regular files differing in size.
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInSize,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			} else if activeActiveModTimeUpdated(srcCtnt, tgtCtnt) {
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInAASourceMTime,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			} else if opts.isMetadata &&
				!metadataEqual(srcCtnt.UserMetadata, tgtCtnt.UserMetadata) &&
				!metadataEqual(srcCtnt.Metadata, tgtCtnt.Metadata) {

				// Regular files user requesting additional metadata to same file.
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInMetadata,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			}

			// No differ
			if returnSimilar {
				diffCh <- diffMessage{
					FirstURL:      srcCtnt.URL.String(),
					SecondURL:     tgtCtnt.URL.String(),
					Diff:          differInNone,
					firstContent:  srcCtnt,
					secondContent: tgtCtnt,
				}
			}
			srcCtnt, srcOk = <-srcCh
			tgtCtnt, tgtOk = <-tgtCh
			continue
		}
		// Differ in second
		diffCh <- diffMessage{
			SecondURL:     tgtCtnt.URL.String(),
			Diff:          differInSecond,
			secondContent: tgtCtnt,
		}
		tgtCtnt, tgtOk = <-tgtCh
		continue
	}

	return nil
}

// objectDifference function finds the difference between all objects
// recursively in sorted order from source and target.
func difference(sourceURL string, sourceCh <-chan *ClientContent, targetURL string, targetCh <-chan *ClientContent, opts mirrorOptions, returnSimilar bool) (diffCh chan diffMessage) {
	diffCh = make(chan diffMessage, 10000)

	go func() {
		defer close(diffCh)

		err := differenceInternal(sourceURL, sourceCh, targetURL, targetCh, opts, returnSimilar, diffCh)
		if err != nil {
			// handle this specifically for filesystem related errors.
			switch v := err.ToGoError().(type) {
			case PathNotFound, PathInsufficientPermission, PathNotADirectory:
				diffCh <- diffMessage{
					Error: err,
				}
				return
			case minio.ErrorResponse:
				switch v.Code {
				case "NoSuchBucket", "NoSuchKey", "SignatureDoesNotMatch":
					diffCh <- diffMessage{
						Error: err,
					}
					return
				}
			}
			errorIf(err, "Unable to list comparison retrying..")
		}
	}()

	return diffCh
}
