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
	"fmt"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	shareUploadFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "recursively upload any object matching the prefix",
		},
		shareFlagExpire,
		shareFlagContentType,
	}
)

// Share documents via URL.
var shareUpload = cli.Command{
	Name:         "upload",
	Usage:        "generate `curl` command to upload objects without requiring access/secret keys",
	Action:       mainShareUpload,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(shareUploadFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Generate a curl command to allow upload access for a single object. Command expires in 7 days (default).
     {{.Prompt}} {{.HelpName}} s3/backup/2006-Mar-1/backup.tar.gz

  2. Generate a curl command to allow upload access to a folder. Command expires in 120 hours.
     {{.Prompt}} {{.HelpName}} --expire=120h s3/backup/2007-Mar-2/

  3. Generate a curl command to allow upload access of only '.png' images to a folder. Command expires in 2 hours.
     {{.Prompt}} {{.HelpName}} --expire=2h --content-type=image/png s3/backup/2007-Mar-2/

  4. Generate a curl command to allow upload access to any objects matching the key prefix 'backup/'. Command expires in 2 hours.
     {{.Prompt}} {{.HelpName}} --recursive --expire=2h s3/backup/2007-Mar-2/backup/
`,
}

// checkShareUploadSyntax - validate command-line args.
func checkShareUploadSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !args.Present() {
		cli.ShowCommandHelpAndExit(ctx, "upload", 1) // last argument is exit code.
	}

	// Set command flags from context.
	isRecursive := ctx.Bool("recursive")
	expireArg := ctx.String("expire")

	// Parse expiry.
	expiry := shareDefaultExpiry
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=`"+expireArg+"`.")
	}

	// Validate expiry.
	if expiry.Seconds() < 1 {
		fatalIf(errDummy().Trace(expiry.String()),
			"Expiry cannot be lesser than 1 second.")
	}
	if expiry.Seconds() > 604800 {
		fatalIf(errDummy().Trace(expiry.String()),
			"Expiry cannot be larger than 7 days.")
	}

	for _, targetURL := range ctx.Args() {
		url := newClientURL(targetURL)
		if strings.HasSuffix(targetURL, string(url.Separator)) && !isRecursive {
			fatalIf(errInvalidArgument().Trace(targetURL),
				"Use --recursive flag to generate curl command for prefixes.")
		}
	}
}

// makeCurlCmd constructs curl command-line.
func makeCurlCmd(key, postURL string, isRecursive bool, uploadInfo map[string]string) (string, *probe.Error) {
	postURL += " "
	curlCommand := "curl " + postURL
	for k, v := range uploadInfo {
		if k == "key" {
			key = v
			continue
		}
		curlCommand += fmt.Sprintf("-F %s=%s ", k, v)
	}
	// If key starts with is enabled prefix it with the output.
	if isRecursive {
		curlCommand += fmt.Sprintf("-F key=%s<NAME> ", key) // Object name.
	} else {
		curlCommand += fmt.Sprintf("-F key=%s ", key) // Object name.
	}
	curlCommand += "-F file=@<FILE>" // File to upload.
	return curlCommand, nil
}

// save shared URL to disk.
func saveSharedURL(objectURL string, shareURL string, expiry time.Duration, contentType string) *probe.Error {
	// Load previously saved upload-shares.
	shareDB := newShareDBV1()
	if err := shareDB.Load(getShareUploadsFile()); err != nil {
		return err.Trace(getShareUploadsFile())
	}

	// Make new entries to uploadsDB.
	shareDB.Set(objectURL, shareURL, expiry, contentType)
	shareDB.Save(getShareUploadsFile())

	return nil
}

// doShareUploadURL uploads files to the target.
func doShareUploadURL(ctx context.Context, objectURL string, isRecursive bool, expiry time.Duration, contentType string) *probe.Error {
	clnt, err := newClient(objectURL)
	if err != nil {
		return err.Trace(objectURL)
	}

	// Generate pre-signed access info.
	shareURL, uploadInfo, err := clnt.ShareUpload(context.Background(), isRecursive, expiry, contentType)
	if err != nil {
		return err.Trace(objectURL, "expiry="+expiry.String(), "contentType="+contentType)
	}

	// Get the new expanded url.
	objectURL = clnt.GetURL().String()

	// Generate curl command.
	curlCmd, err := makeCurlCmd(objectURL, shareURL, isRecursive, uploadInfo)
	if err != nil {
		return err.Trace(objectURL)
	}

	printMsg(shareMesssage{
		ObjectURL:   objectURL,
		ShareURL:    curlCmd,
		TimeLeft:    expiry,
		ContentType: contentType,
	})

	// save shared URL to disk.
	return saveSharedURL(objectURL, curlCmd, expiry, contentType)
}

// main for share upload command.
func mainShareUpload(cliCtx *cli.Context) error {
	ctx, cancelShareDownload := context.WithCancel(globalContext)
	defer cancelShareDownload()

	// check input arguments.
	checkShareUploadSyntax(cliCtx)

	// Initialize share config folder.
	initShareConfig()

	// Additional command speific theme customization.
	shareSetColor()

	// Set command flags from context.
	isRecursive := cliCtx.Bool("recursive")
	expireArg := cliCtx.String("expire")
	expiry := shareDefaultExpiry
	contentType := cliCtx.String("content-type")
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=`"+expireArg+"`.")
	}

	for _, targetURL := range cliCtx.Args() {
		err := doShareUploadURL(ctx, targetURL, isRecursive, expiry, contentType)
		if err != nil {
			switch err.ToGoError().(type) {
			case APINotImplemented:
				fatalIf(err.Trace(), "Unable to share a non S3 url `"+targetURL+"`.")
			default:
				fatalIf(err.Trace(targetURL), "Unable to generate curl command for upload `"+targetURL+"`.")
			}
		}
	}
	return nil
}
