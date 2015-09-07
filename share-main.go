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
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Share documents via URL.
var shareCmd = cli.Command{
	Name:   "share",
	Usage:  "Share documents via URL.",
	Action: mainShare,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [DURATION]

   DURATION = NN[h|m|s] [DEFAULT=168h]

EXAMPLES:
   1. Generate URL for sharing, with a default expiry of 7 days.
      $ mc {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz

   2. Generate URL for sharing, with an expiry of 10 minutes.
      $ mc {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz 10m

   3. Generate list of URLs for sharing a folder recursively, with expiration of 5 days each.
      $ mc {{.Name}} https://s3.amazonaws.com/backup... 120h

   4. Generate URL with space characters for sharing, with an expiry of 5 seconds.
      $ mc {{.Name}} s3/miniocloud/nothing-like-anything 5s

`,
}

func checkShareSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
	}
	if len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
	}
}

// mainShare - is a handler for mc share command
func mainShare(ctx *cli.Context) {
	checkShareSyntax(ctx)

	console.SetCustomTheme(map[string]*color.Color{
		"Share": color.New(color.FgGreen, color.Bold),
	})

	args := ctx.Args()
	config := mustGetMcConfig()

	/// get first and last arguments
	url := args.First() // url to be shared
	// default expiration is 7days
	expires := time.Duration(604800) * time.Second
	if len(args) == 2 {
		var err error
		expires, err = time.ParseDuration(args.Last())
		fatalIf(probe.NewError(err), "Unable to parse time argument.")
	}

	targetURL := getAliasURL(url, config.Aliases)

	// if recursive strip off the "..."
	err := doShareCmd(stripRecursiveURL(targetURL), isURLRecursive(targetURL), shareDuration{duration: expires})
	fatalIf(err.Trace(targetURL), "Unable to generate URL for sharing.")
}

func getNewTargetURL(targetParser *client.URL, name string) string {
	match, _ := filepath.Match("*.s3*.amazonaws.com", targetParser.Host)
	if match {
		targetParser.Path = string(targetParser.Separator) + name
	} else {
		targetParser.Path = string(targetParser.Separator) + path2Bucket(targetParser) + string(targetParser.Separator) + name
	}
	return targetParser.String()
}

// doShareCmd share files from target
func doShareCmd(targetURL string, recursive bool, expires shareDuration) *probe.Error {
	clnt, err := target2Client(targetURL)
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
		newClnt, err := url2Client(getNewTargetURL(clnt.URL(), contentCh.Content.Name))
		if err != nil {
			return err.Trace()
		}
		presignedURL, err := newClnt.Share(expires.GetDuration())
		if err != nil {
			return err.Trace()
		}
		expires.presignedURL = presignedURL
		console.Println(expires)
	}
	return nil
}

// this code is necessary since, share only operates on cloud storage URLs not filesystem
func path2Bucket(u *client.URL) (bucketName string) {
	pathSplits := strings.SplitN(u.Path, "?", 2)
	splits := strings.SplitN(pathSplits[0], string(u.Separator), 3)
	switch len(splits) {
	case 0, 1:
		bucketName = ""
	case 2:
		bucketName = splits[1]
	case 3:
		bucketName = splits[1]
	}
	return bucketName
}
