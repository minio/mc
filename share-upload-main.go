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
	"github.com/minio/minio/pkg/probe"
)

var (
	shareUploadFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of share download.",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "Recursively upload any object matching the prefix.",
		},
		shareFlagExpire,
		shareFlagContentType,
	}
)

// Share documents via URL.
var shareUpload = cli.Command{
	Name:   "upload",
	Usage:  "Generate ‘curl’ command to upload objects without requiring access/secret keys.",
	Action: mainShareUpload,
	Flags:  append(shareUploadFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc share {{.Name}} - {{.Usage}}

USAGE:
   mc share {{.Name}} [OPTIONS] TARGET [TARGET...]

OPTIONS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Generate a curl command to allow upload access for a single object. Command expires in 7 days (default).
      $ mc share {{.Name}} s3/backup/2006-Mar-1/backup.tar.gz

   2. Generate a curl command to allow upload access to a folder. Command expires in 120 hours.
      $ mc share {{.Name}} --expire=120h s3/backup/2007-Mar-2/

   3. Generate a curl command to allow upload access of only '.png' images to a folder. Command expires in 2 hours.
      $ mc share {{.Name}} --expire=2h --content-type=image/png s3/backup/2007-Mar-2/

   4. Generate a curl command to allow upload access to any objects matching the key prefix 'backup/'. Command expires in 2 hours.
      $ mc share {{.Name}} --recursive --expire=2h s3/backup/2007-Mar-2/backup/
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
		fatalIf(probe.NewError(e), "Unable to parse expire=‘"+expireArg+"’.")
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
				"Use --recursive option to generate curl command for prefixes.")
		}
	}
}

// makeCurlCmd constructs curl command-line.
func makeCurlCmd(key string, isRecursive bool, uploadInfo map[string]string) string {
	URL := newClientURL(key)
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
	// If key starts with is enabled prefix it with the output.
	if isRecursive {
		curlCommand += fmt.Sprintf("-F key=%s<NAME> ", key) // Object name.
	} else {
		curlCommand += fmt.Sprintf("-F key=%s ", key) // Object name.
	}
	curlCommand += "-F file=@<FILE>" // File to upload.
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
func doShareUploadURL(objectURL string, isRecursive bool, expiry time.Duration, contentType string) *probe.Error {
	clnt, err := newClient(objectURL)
	if err != nil {
		return err.Trace(objectURL)
	}

	// Generate pre-signed access info.
	uploadInfo, err := clnt.ShareUpload(isRecursive, expiry, contentType)
	if err != nil {
		return err.Trace(objectURL, "expiry="+expiry.String(), "contentType="+contentType)
	}

	// Get the new expanded url.
	objectURL = clnt.GetURL().String()

	// Generate curl command.
	curlCmd := makeCurlCmd(objectURL, isRecursive, uploadInfo)

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
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check input arguments.
	checkShareUploadSyntax(ctx)

	// Initialize share config folder.
	initShareConfig()

	// Additional command speific theme customization.
	shareSetColor()

	// Set command flags from context.
	isRecursive := ctx.Bool("recursive")
	expireArg := ctx.String("expire")
	expiry := shareDefaultExpiry
	contentType := ctx.String("content-type")
	if expireArg != "" {
		var e error
		expiry, e = time.ParseDuration(expireArg)
		fatalIf(probe.NewError(e), "Unable to parse expire=‘"+expireArg+"’.")
	}

	for _, targetURL := range ctx.Args() {
		err := doShareUploadURL(targetURL, isRecursive, expiry, contentType)
		fatalIf(err.Trace(targetURL), "Unable to generate curl command for upload ‘"+targetURL+"’.")
	}
}
