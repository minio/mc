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
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
)

var (
	shareFlagUploadHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of share download.",
	}
)

// Share documents via URL.
var shareUpload = cli.Command{
	Name:   "upload",
	Usage:  "Generate ‘curl’ command to upload objects without requiring access/secret keys.",
	Action: mainShareUpload,
	Flags:  []cli.Flag{shareFlagExpire, shareFlagContentType, shareFlagUploadHelp},
	CustomHelpTemplate: `NAME:
   mc share {{.Name}} - {{.Usage}}

USAGE:
   mc share {{.Name}} [OPTIONS] TARGET [TARGET...]

OPTIONS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Generate a curl command to allow upload access for a single object. Command expires after 7 days (default).
      $ mc share {{.Name}} s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz

   2. Generate a curl command to allow upload access to a folder. Command expires in 120 hours.
      $ mc share {{.Name}} --expire=120h s3.amazonaws.com/backup/2007-Mar-2/...

   3. Generate a curl command to allow upload access of only '.png' images to a folder. Command expires in 2 hours.
      $ mc share {{.Name}} --expire=2h --content-type=image/png s3.amazonaws.com/backup/2007-Mar-2/...
`,
}

// checkShareUploadSyntax - validate command-line args.
func checkShareUploadSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !args.Present() {
		cli.ShowCommandHelpAndExit(ctx, "upload", 1) // last argument is exit code.
	}

	// Parse expiry.
	// isRecursive := ctx.Bool("recursive")
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

	URLs, err := args2URLs(ctx.Args()) // expand alias.
	fatalIf(err.Trace(ctx.Args()...), "Unable to convert args to URLs.")

	for _, url := range URLs {
		_, _, err := url2Stat(url)
		fatalIf(err.Trace(url), "Unable to stat ‘"+url+"’.")
	}
}

// makeCurlCmd constructs curl command-line.
func makeCurlCmd(key string, uploadInfo map[string]string) string {
	URL := client.NewURL(key)
	postURL := URL.Scheme + URL.SchemeSeparator + URL.Host + string(URL.Separator)
	if !isBucketVirtualStyle(URL.Host) {
		postURL = postURL + uploadInfo["bucket"]
	}
	postURL += " "
	curlCommand := "curl " + postURL
	for k, v := range uploadInfo {
		if k == "key" {
			key = v
			continue
		}
		curlCommand += fmt.Sprintf("-F %s=%s ", k, v)
	}
	curlCommand += fmt.Sprintf("-F key=%s ", key) // Object name.
	curlCommand += "-F file=@<FILE>"              // File to upload.
	return curlCommand
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
func doShareUploadURL(objectURL string, recursive bool, expiry time.Duration, contentType string) *probe.Error {
	clnt, err := url2Client(objectURL)
	if err != nil {
		return err.Trace(objectURL)
	}

	// Generate pre-signed access info.
	uploadInfo, err := clnt.ShareUpload(recursive, expiry, contentType)
	if err != nil {
		return err.Trace(objectURL, "expiry="+expiry.String(), "contentType="+contentType)
	}

	// Generate curl command.
	curlCmd := makeCurlCmd(objectURL, uploadInfo)

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
func mainShareUpload(ctx *cli.Context) {
	// Initialize share config folder.
	initShareConfig()

	// Additional command speific theme customization.
	shareSetColor()

	// check input arguments.
	checkShareUploadSyntax(ctx)

	isRecursive := ctx.Bool("recursive")
	expireArg := ctx.String("expire")
	expiry := shareDefaultExpiry
	contentType := ctx.String("content-type")
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=‘"+expireArg+"’.")
	}

	URLs, err := args2URLs(ctx.Args()) // expand alias.
	fatalIf(err.Trace(ctx.Args()...), "Unable to convert args to URLs.")

	for _, targetURL := range URLs {
		err := doShareUploadURL(targetURL, isRecursive, expiry, contentType)
		fatalIf(err.Trace(targetURL), "Unable to generate curl command for upload ‘"+targetURL+"’.")
	}
}
