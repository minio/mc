/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"time"

	"github.com/minio/cli"
	"github.com/minio/minio-xl/pkg/probe"
)

// Share documents via URL.
var shareDownload = cli.Command{
	Name:   "download",
	Usage:  "Generate URLs for download access.",
	Action: mainShareDownload,
	Flags:  []cli.Flag{shareFlagExpire},
	CustomHelpTemplate: `NAME:
   mc share {{.Name}} - {{.Usage}}

USAGE:
   mc share {{.Name}} [OPTIONS] TARGET

OPTIONS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Generate URL for sharing, with a default expiry of 7 days.
      $ mc share {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz

   2. Generate URL for sharing, with an expiry of 10 minutes.
      $ mc share {{.Name}} --expire=10m https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz

   3. Generate list of URLs for sharing a folder recursively, with an expiry of 5 days each.
      $ mc share {{.Name}} --expire=120h https://s3.amazonaws.com/backup...
`,
}

// checkShareDownloadSyntax - validate command-line args.
func checkShareDownloadSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !args.Present() {
		cli.ShowCommandHelpAndExit(ctx, "download", 1) // last argument is exit code.
	}

	if !isURLRecursive(args.First()) {
		url := stripRecursiveURL(args.First())
		if strings.HasSuffix(url, "/") {
			fatalIf(errDummy().Trace(), fmt.Sprintf("To grant access to an entire folder, you may use ‘%s’.", url+recursiveSeparator))
		}
	}

	// Parse expiry.
	expiry := shareDefaultExpiry
	expireArg := ctx.String("expire")
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=‘"+expireArg+"’.")
	}

	// Validate expiry.
	if expiry.Seconds() < 1 {
		fatalIf(errDummy().Trace(expiry.String()), "Expiry cannot be lesser than 1 second.")
	}
	if expiry.Seconds() > 604800 {
		fatalIf(errDummy().Trace(expiry.String()), "Expiry cannot be larger than 7 days.")
	}
}

// doShareURL share files from target
func doShareDownloadURL(targetURL string, recursive bool, expiry time.Duration) *probe.Error {
	clnt, err := url2Client(targetURL)
	if err != nil {
		return err.Trace()
	}

	// Load previously saved upload-shares. Add new entries and write it back.
	shareDB := newShareDBV1()
	err = shareDB.Load(getShareDownloadsFile())
	if err != nil {
		return err.Trace()
	}

	// Generate share URL for each target.
	incomplete := false
	for content := range clnt.List(recursive, incomplete) {
		if content.Err != nil {
			return content.Err.Trace()
		}

		objectURL := content.Content.URL.String()
		newClnt, err := url2Client(objectURL)
		if err != nil {
			return err.Trace()
		}

		// Generate share URL.
		shareURL, err := newClnt.ShareDownload(expiry)
		if err != nil {
			return err.Trace()
		}

		// Make new entries to shareDB.
		contentType := "" // Not useful for download shares.
		shareDB.Set(objectURL, shareURL, expiry, contentType)
		printMsg(shareMesssage{
			ObjectURL:   objectURL,
			ShareURL:    shareURL,
			TimeLeft:    expiry,
			ContentType: contentType,
		})
	}

	// Save downloads and return.
	return shareDB.Save(getShareDownloadsFile())
}

// main for share download.
func mainShareDownload(ctx *cli.Context) {
	// Initialize share config folder.
	initShareConfig()

	// Additional command speific theme customization.
	shareSetColor()

	// check input arguments.
	checkShareDownloadSyntax(ctx)

	// Extract arguments.
	config := mustGetMcConfig()
	args := ctx.Args()
	url := stripRecursiveURL(args.First())
	isRecursive := isURLRecursive(args.First())
	expiry := shareDefaultExpiry
	expireArg := ctx.String("expire")
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=‘"+expireArg+"’.")
	}

	targetURL := getAliasURL(stripRecursiveURL(url), config.Aliases) // Expand alias.

	// if recursive strip off the "..."
	err := doShareDownloadURL(stripRecursiveURL(targetURL), isRecursive, expiry)
	fatalIf(err.Trace(targetURL), "Unable to share target ‘"+args.First()+"’.")
}
