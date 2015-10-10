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
	"errors"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

// Share documents via URL.
var shareDownload = cli.Command{
	Name:   "download",
	Usage:  "Share documents via URL.",
	Action: mainShareDownload,
	CustomHelpTemplate: `NAME:
    mc share {{.Name}} - {{.Usage}}

 USAGE:
    mc share {{.Name}} download TARGET [DURATION]

    DURATION = NN[h|m|s] [DEFAULT=168h]

 EXAMPLES:
    1. Generate URL for sharing, with a default expiry of 7 days.
       $ mc share {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz

    2. Generate URL for sharing, with an expiry of 10 minutes.
       $ mc share {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz 10m

    3. Generate list of URLs for sharing a folder recursively, with expiration of 5 days each.
       $ mc share {{.Name}} https://s3.amazonaws.com/backup... 120h

    4. Generate URL with space characters for sharing, with an expiry of 5 seconds.
       $ mc share {{.Name}} s3/miniocloud/nothing-like-anything 5s

`,
}

func checkShareDownloadSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !args.Present() || args.First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "download", 1) // last argument is exit code
	}
	if len(args) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "download", 1) // last argument is exit code
	}
}

func mainShareDownload(ctx *cli.Context) {
	shareDataDirSetup()
	checkShareDownloadSyntax(ctx)
	args := ctx.Args()
	config := mustGetMcConfig()
	url := args.Get(0)
	// default expiration is 7days
	expires := time.Duration(604800) * time.Second
	if len(args) == 2 {
		var err error
		expires, err = time.ParseDuration(args.Get(1))
		fatalIf(probe.NewError(err), "Unable to parse time argument.")
	}

	targetURL := getAliasURL(url, config.Aliases)

	setSharePalette(ctx.GlobalString("colors"))

	// if recursive strip off the "..."
	err := doShareDownloadURL(stripRecursiveURL(targetURL), isURLRecursive(targetURL), expires)
	fatalIf(err.Trace(targetURL), "Unable to generate URL for download.")
	return
}

// doShareURL share files from target
func doShareDownloadURL(targetURL string, recursive bool, expires time.Duration) *probe.Error {
	shareDate := time.Now().UTC()
	sURLs, err := loadSharedURLsV3()
	if err != nil {
		return err.Trace()
	}
	var clnt client.Client
	clnt, err = target2Client(targetURL)
	if err != nil {
		return err.Trace()
	}
	if expires.Seconds() < 1 {
		return probe.NewError(errors.New("Too low expires, expiration cannot be less than 1 second."))
	}
	if expires.Seconds() > 604800 {
		return probe.NewError(errors.New("Too high expires, expiration cannot be larger than 7 days."))
	}
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			return contentCh.Err.Trace()
		}
		var newClnt client.Client
		newClnt, err = url2Client(getNewTargetURL(clnt.URL(), contentCh.Content.Name))
		if err != nil {
			return err.Trace()
		}
		var sharedURL string
		sharedURL, err = newClnt.ShareDownload(expires)
		if err != nil {
			return err.Trace()
		}
		shareMessage := ShareMessage{
			Expiry:      expires,
			DownloadURL: sharedURL,
			Key:         newClnt.URL().String(),
		}
		shareMessageV3 := ShareMessageV3{
			Expiry:      expires,
			DownloadURL: sharedURL,
			Key:         newClnt.URL().String(),
		}
		sURLs.URLs = append(sURLs.URLs, struct {
			Date    time.Time
			Message ShareMessageV3
		}{
			Date:    shareDate,
			Message: shareMessageV3,
		})
		Prints("%s\n", shareMessage)
	}
	saveSharedURLsV3(sURLs)
	return nil
}
