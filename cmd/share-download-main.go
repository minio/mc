/*
 * MinIO Client (C) 2014, 2015 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	shareDownloadFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "share all objects recursively",
		},
		shareFlagExpire,
	}
)

// Share documents via URL.
var shareDownload = cli.Command{
	Name:   "download",
	Usage:  "generate URLs for download access",
	Action: mainShareDownload,
	Before: setGlobalsFromContext,
	Flags:  append(shareDownloadFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Share this object with 7 days default expiry.
      $ {{.HelpName}} s3/backup/2006-Mar-1/backup.tar.gz

   2. Share this object with 10 minutes expiry.
      $ {{.HelpName}} --expire=10m s3/backup/2006-Mar-1/backup.tar.gz

   3. Share all objects under this folder with 5 days expiry.
      $ {{.HelpName}} --expire=120h s3/backup/2006-Mar-1/

   4. Share all objects under this bucket and all its folders and sub-folders with 5 days expiry.
      $ {{.HelpName}} --recursive --expire=120h s3/backup/

`,
}

// checkShareDownloadSyntax - validate command-line args.
func checkShareDownloadSyntax(ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	args := ctx.Args()
	if !args.Present() {
		cli.ShowCommandHelpAndExit(ctx, "download", 1) // last argument is exit code.
	}

	// Parse expiry.
	expiry := shareDefaultExpiry
	expireArg := ctx.String("expire")
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=`"+expireArg+"`.")
	}

	// Validate expiry.
	if expiry.Seconds() < 1 {
		fatalIf(errDummy().Trace(expiry.String()), "Expiry cannot be lesser than 1 second.")
	}
	if expiry.Seconds() > 604800 {
		fatalIf(errDummy().Trace(expiry.String()), "Expiry cannot be larger than 7 days.")
	}

	// Validate if object exists only if the `--recursive` flag was NOT specified
	isRecursive := ctx.Bool("recursive")
	if !isRecursive {
		for _, url := range ctx.Args() {
			_, _, err := url2Stat(url, false, encKeyDB)
			if err != nil {
				fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
			}
		}
	}
}

// doShareURL share files from target.
func doShareDownloadURL(targetURL string, isRecursive bool, expiry time.Duration) *probe.Error {
	targetAlias, targetURLFull, _, err := expandAlias(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	clnt, err := newClientFromAlias(targetAlias, targetURLFull)
	if err != nil {
		return err.Trace(targetURL)
	}

	// Load previously saved upload-shares. Add new entries and write it back.
	shareDB := newShareDBV1()
	shareDownloadsFile := getShareDownloadsFile()
	err = shareDB.Load(shareDownloadsFile)
	if err != nil {
		return err.Trace(shareDownloadsFile)
	}

	// Generate share URL for each target.
	isIncomplete := false
	isFetchMeta := false
	// Channel which will receive objects whose URLs need to be shared
	objectsCh := make(chan *clientContent)

	content, err := clnt.Stat(isIncomplete, isFetchMeta, nil)
	if err != nil {
		return err.Trace(clnt.GetURL().String())
	}

	if !content.Type.IsDir() {
		go func() {
			defer close(objectsCh)
			objectsCh <- content
		}()
	} else {
		if !strings.HasSuffix(targetURLFull, string(clnt.GetURL().Separator)) {
			targetURLFull = targetURLFull + string(clnt.GetURL().Separator)
		}
		clnt, err = newClientFromAlias(targetAlias, targetURLFull)
		if err != nil {
			return err.Trace(targetURLFull)
		}
		// Recursive mode: Share list of objects
		go func() {
			defer close(objectsCh)
			for content := range clnt.List(isRecursive, isIncomplete, DirNone) {
				objectsCh <- content
			}
		}()
	}

	// Iterate over all objects to generate share URL
	for content := range objectsCh {
		if content.Err != nil {
			return content.Err.Trace(clnt.GetURL().String())
		}
		// if any incoming directories, we don't need to calculate.
		if content.Type.IsDir() {
			continue
		}
		objectURL := content.URL.String()
		newClnt, err := newClientFromAlias(targetAlias, objectURL)
		if err != nil {
			return err.Trace(objectURL)
		}

		// Generate share URL.
		shareURL, err := newClnt.ShareDownload(expiry)
		if err != nil {
			// add objectURL and expiry as part of the trace arguments.
			return err.Trace(objectURL, "expiry="+expiry.String())
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
	return shareDB.Save(shareDownloadsFile)
}

// main for share download.
func mainShareDownload(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check input arguments.
	checkShareDownloadSyntax(ctx, encKeyDB)

	// Initialize share config folder.
	initShareConfig()

	// Additional command speific theme customization.
	shareSetColor()

	// Set command flags from context.
	isRecursive := ctx.Bool("recursive")
	expiry := shareDefaultExpiry
	if ctx.String("expire") != "" {
		var e error
		expiry, e = time.ParseDuration(ctx.String("expire"))
		fatalIf(probe.NewError(e), "Unable to parse expire=`"+ctx.String("expire")+"`.")
	}

	for _, targetURL := range ctx.Args() {
		err := doShareDownloadURL(targetURL, isRecursive, expiry)
		if err != nil {
			switch err.ToGoError().(type) {
			case APINotImplemented:
				fatalIf(err.Trace(), "Unable to share a non S3 url `"+targetURL+"`.")
			default:
				fatalIf(err.Trace(targetURL), "Unable to share target `"+targetURL+"`.")
			}
		}
	}
	return nil
}
