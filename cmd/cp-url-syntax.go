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
	"fmt"
	"runtime"
	"time"

	"github.com/minio/cli"
)

func checkCopySyntax(ctx context.Context, cliCtx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	if len(cliCtx.Args()) < 2 {
		showCommandHelpAndExit(cliCtx, 1) // last argument is exit code.
	}

	// extract URLs.
	URLs := cliCtx.Args()
	if len(URLs) < 2 {
		fatalIf(errDummy().Trace(cliCtx.Args()...), "Unable to parse source and target arguments.")
	}

	srcURLs := URLs[:len(URLs)-1]
	tgtURL := URLs[len(URLs)-1]
	isZip := cliCtx.Bool("zip")
	versionID := cliCtx.String("version-id")

	if versionID != "" && len(srcURLs) > 1 {
		fatalIf(errDummy().Trace(cliCtx.Args()...), "Unable to pass --version flag with multiple copy sources arguments.")
	}

	if isZip && cliCtx.String("rewind") != "" {
		fatalIf(errDummy().Trace(cliCtx.Args()...), "--zip and --rewind cannot be used together")
	}

	// Check if bucket name is passed for URL type arguments.
	url := newClientURL(tgtURL)
	if url.Host != "" {
		if url.Path == string(url.Separator) {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Target `%s` does not contain bucket name.", tgtURL))
		}
	}

	if cliCtx.String(rdFlag) != "" && cliCtx.String(rmFlag) == "" {
		fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Both object retention flags `--%s` and `--%s` are required.\n", rdFlag, rmFlag))
	}

	if cliCtx.String(rdFlag) == "" && cliCtx.String(rmFlag) != "" {
		fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Both object retention flags `--%s` and `--%s` are required.\n", rdFlag, rmFlag))
	}

	// Preserve functionality not supported for windows
	if cliCtx.Bool("preserve") && runtime.GOOS == "windows" {
		fatalIf(errInvalidArgument().Trace(), "Permissions are not preserved on windows platform.")
	}
}

// checkCopySyntaxTypeA verifies if the source and target are valid file arguments.
func checkCopySyntaxTypeA(ctx context.Context, srcURL, versionID string, keys map[string][]prefixSSEPair, isZip bool, timeRef time.Time) {
	_, srcContent, err := url2Stat(ctx, srcURL, versionID, false, keys, timeRef, isZip)
	fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")

	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(), "Source `"+srcURL+"` is not a file.")
	}
}

// checkCopySyntaxTypeB verifies if the source is a valid file and target is a valid folder.
func checkCopySyntaxTypeB(ctx context.Context, srcURL, versionID, tgtURL string, keys map[string][]prefixSSEPair, isZip bool, timeRef time.Time) {
	_, srcContent, err := url2Stat(ctx, srcURL, versionID, false, keys, timeRef, isZip)
	fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")

	if !srcContent.Type.IsRegular() {
		fatalIf(errInvalidArgument().Trace(srcURL), "Source `"+srcURL+"` is not a file.")
	}

	// Check target.
	if _, tgtContent, err := url2Stat(ctx, tgtURL, "", false, keys, timeRef, false); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target `"+tgtURL+"` is not a folder.")
		}
	}
}

// checkCopySyntaxTypeC verifies if the source is a valid recursive dir and target is a valid folder.
func checkCopySyntaxTypeC(ctx context.Context, srcURLs []string, tgtURL string, isRecursive, isZip bool, keys map[string][]prefixSSEPair, isMvCmd bool, timeRef time.Time) {
	// Check source.
	if len(srcURLs) != 1 {
		fatalIf(errInvalidArgument().Trace(), "Invalid number of source arguments.")
	}

	// Check target.
	if _, tgtContent, err := url2Stat(ctx, tgtURL, "", false, keys, timeRef, false); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target `"+tgtURL+"` is not a folder.")
		}
	}

	for _, srcURL := range srcURLs {
		c, srcContent, err := url2Stat(ctx, srcURL, "", false, keys, timeRef, isZip)
		fatalIf(err.Trace(srcURL), "Unable to stat source `"+srcURL+"`.")

		if srcContent.Type.IsDir() {
			// Require --recursive flag if we are copying a directory
			if !isRecursive {
				operation := "copy"
				if isMvCmd {
					operation = "move"
				}
				fatalIf(errInvalidArgument().Trace(srcURL), fmt.Sprintf("To %v a folder requires --recursive flag.", operation))
			}

			// Check if we are going to copy a directory into itself
			if isURLContains(srcURL, tgtURL, string(c.GetURL().Separator)) {
				operation := "Copying"
				if isMvCmd {
					operation = "Moving"
				}
				fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%v a folder into itself is not allowed.", operation))
			}
		}
	}
}

// checkCopySyntaxTypeD verifies if the source is a valid list of files and target is a valid folder.
func checkCopySyntaxTypeD(ctx context.Context, tgtURL string, keys map[string][]prefixSSEPair, timeRef time.Time) {
	// Source can be anything: file, dir, dir...
	// Check target if it is a dir
	if _, tgtContent, err := url2Stat(ctx, tgtURL, "", false, keys, timeRef, false); err == nil {
		if !tgtContent.Type.IsDir() {
			fatalIf(errInvalidArgument().Trace(tgtURL), "Target `"+tgtURL+"` is not a folder.")
		}
	}
}
