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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// Help message.
var shareCmd = cli.Command{
	Name:   "share",
	Usage:  "Share presigned URLs from cloud storage",
	Action: runShareCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET DURATION {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Generate presigned url for an object, with expiration of 10minutes
      $ mc {{.Name}} url https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz 10m

   2. Generate presigned url for all objects at given path, with expiration of 20minutes
      $ mc {{.Name}} url https://s3.amazonaws.com/backup... 20m

`,
}

// runShareCmd - is a handler for mc share command
func runShareCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
	}
	args := ctx.Args()
	config := mustGetMcConfig()
	/// get first and last arguments
	url := args.First() // url to be shared
	// default expiration is 7days
	expires := time.Duration(604800) * time.Second
	if len(args) == 2 {
		var err error
		expires, err = time.ParseDuration(args.Last())
		Fatal(probe.New(err))
	}
	targetURL, err := getExpandedURL(url, config.Aliases)
	Fatal(err)
	// if recursive strip off the "..."
	newTargetURL := stripRecursiveURL(targetURL)
	Fatal(doShareCmd(newTargetURL, isURLRecursive(targetURL), expires))
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
func doShareCmd(targetURL string, recursive bool, expires time.Duration) *probe.Error {
	clnt, err := target2Client(targetURL)
	if err != nil {
		return err.Trace()
	}
	if expires.Seconds() < 1 {
		return probe.New(errors.New("Too low expires, expiration cannot be less than 1 second"))
	}
	if expires.Seconds() > 604800 {
		return probe.New(errors.New("Too high expires, expiration cannot be larger than 7 days"))
	}
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			return contentCh.Err.Trace()
		}
		newClnt, err := url2Client(getNewTargetURL(clnt.URL(), contentCh.Content.Name))
		if err != nil {
			return err.Trace()
		}
		presignedURL, err := newClnt.PresignedGetObject(expires, 0, 0)
		if err != nil {
			return err.Trace()
		}
		console.PrintC(fmt.Sprintf("Succesfully generated shared URL with expiry %s, please copy: %s\n", expires, presignedURL))
	}
	return nil
}

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
