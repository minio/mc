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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var shareCmd = cli.Command{
	Name:   "share",
	Usage:  "Share documents via URL.",
	Action: mainShare,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [DURATION=s|m|h|d] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Generate URL for sharing, with a default expiry of 7 days.
      $ mc {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz

   2. Generate URL for sharing, with an expiry of 10 minutes.
      $ mc {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz 10m

   3. Generate list of URLs for sharing a folder recursively, with expiration of 1 hour each.
      $ mc {{.Name}} https://s3.amazonaws.com/backup... 1h

`,
}

// mainShare - is a handler for mc share command
func mainShare(ctx *cli.Context) {
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
		fatalIf(probe.NewError(err), "Unable to parse time argument.")
	}
	targetURL, err := getCanonicalizedURL(url, config.Aliases)
	fatalIf(err.Trace(url), "Unable to parse argument ‘"+url+"’.")

	// if recursive strip off the "..."
	err = doShareCmd(stripRecursiveURL(targetURL), isURLRecursive(targetURL), expires)
	fatalIf(err.Trace(targetURL), "Unable generate URL for sharing.")
}

// ShareMessage container for share messages
type ShareMessage struct {
	Expires      time.Duration `json:"expire-seconds"`
	PresignedURL string        `json:"presigned-url"`
}

// String string printer for share message
func (s ShareMessage) String() string {
	if !globalJSONFlag {
		return fmt.Sprintf("Succesfully generated shared URL with expiry %s, please share: %s", s.Expires, s.PresignedURL)
	}
	shareMessageBytes, err := json.Marshal(s)
	fatalIf(probe.NewError(err), "Failed to marshal into JSON.")

	// json encoding escapes ampersand into its unicode character which is not usable directly for share
	// and fails with cloud storage. convert them back so that they are usable
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	return string(shareMessageBytes)
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
		presignedURL, err := newClnt.Share(expires)
		if err != nil {
			return err.Trace()
		}
		console.PrintC(ShareMessage{Expires: expires, PresignedURL: presignedURL}.String() + "\n")
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
