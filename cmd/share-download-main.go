// Copyright (c) 2015-2021 MinIO, Inc.
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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	shareDownloadFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "share all objects recursively",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "share a particular object version",
		},
		shareFlagExpire,
	}
)

// Share documents via URL.
var shareDownload = cli.Command{
	Name:         "download",
	Usage:        "generate URLs for download access",
	Action:       mainShareDownload,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(shareDownloadFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Share this object with 7 days default expiry.
     {{.Prompt}} {{.HelpName}} s3/backup/2006-Mar-1/backup.tar.gz

  2. Share this object with 10 minutes expiry.
     {{.Prompt}} {{.HelpName}} --expire=10m s3/backup/2006-Mar-1/backup.tar.gz

  3. Share all objects under this folder with 5 days expiry.
     {{.Prompt}} {{.HelpName}} --expire=120h s3/backup/2006-Mar-1/

  4. Share all objects under this bucket and all its folders and sub-folders with 5 days expiry.
     {{.Prompt}} {{.HelpName}} --recursive --expire=120h s3/backup/
`,
}

// checkShareDownloadSyntax - validate command-line args.
func checkShareDownloadSyntax(ctx context.Context, cliCtx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	args := cliCtx.Args()
	if !args.Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "download", 1) // last argument is exit code.
	}

	// Parse expiry.
	expiry := shareDefaultExpiry
	expireArg := cliCtx.String("expire")
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

	isRecursive := cliCtx.Bool("recursive")

	versionID := cliCtx.String("version-id")
	if versionID != "" && isRecursive {
		fatalIf(errDummy().Trace(), "--version-id cannot be specified with --recursive flag.")
	}

	// Validate if object exists only if the `--recursive` flag was NOT specified
	if !isRecursive {
		for _, url := range cliCtx.Args() {
			_, _, err := url2Stat(ctx, url, "", false, encKeyDB, time.Time{})
			if err != nil {
				fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
			}
		}
	}
}

// doShareURL share files from target.
func doShareDownloadURL(ctx context.Context, targetURL, versionID string, isRecursive bool, expiry time.Duration) *probe.Error {
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

	// Channel which will receive objects whose URLs need to be shared
	objectsCh := make(chan *ClientContent)

	content, err := clnt.Stat(ctx, StatOptions{versionID: versionID})
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
			for content := range clnt.List(ctx, ListOptions{Recursive: isRecursive, ShowDir: DirNone}) {
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
		objectVersionID := content.VersionID
		newClnt, err := newClientFromAlias(targetAlias, objectURL)
		if err != nil {
			return err.Trace(objectURL)
		}

		// Generate share URL.
		shareURL, err := newClnt.ShareDownload(ctx, objectVersionID, expiry)
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
func mainShareDownload(cliCtx *cli.Context) error {
	ctx, cancelShareDownload := context.WithCancel(globalContext)
	defer cancelShareDownload()

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check input arguments.
	checkShareDownloadSyntax(ctx, cliCtx, encKeyDB)

	// Initialize share config folder.
	initShareConfig()

	// Additional command speific theme customization.
	shareSetColor()

	// Set command flags from context.
	isRecursive := cliCtx.Bool("recursive")
	versionID := cliCtx.String("version-id")
	expiry := shareDefaultExpiry
	if cliCtx.String("expire") != "" {
		var e error
		expiry, e = time.ParseDuration(cliCtx.String("expire"))
		fatalIf(probe.NewError(e), "Unable to parse expire=`"+cliCtx.String("expire")+"`.")
	}

	for _, targetURL := range cliCtx.Args() {
		err := doShareDownloadURL(ctx, targetURL, versionID, isRecursive, expiry)
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
