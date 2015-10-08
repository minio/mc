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

   5. List all the shared URLs which have not expired yet.
      $ mc {{.Name}} list

`,
}

// ShareMessage is container for share command on success and failure messages
type ShareMessage struct {
	Expiry time.Duration `json:"expiry"`
	URL    string        `json:"url"`
	Key    string        `json:"keyName"`
}

// String - regular colorized message
func (s ShareMessage) String() string {
	return console.Colorize("Share", fmt.Sprintf("Expiry: %s\n   URL: %s\n   Key: %s", timeDurationToHumanizedTime(s.Expiry), s.URL, s.Key))
}

// JSON json message for share command
func (s ShareMessage) JSON() string {
	shareMessageBytes, err := json.Marshal(struct {
		Expiry humanizedTime `json:"expiry"`
		URL    string        `json:"url"`
		Key    string        `json:"keyName"`
	}{
		Expiry: timeDurationToHumanizedTime(s.Expiry),
		URL:    s.URL,
		Key:    s.Key,
	})
	fatalIf(probe.NewError(err), "Failed to marshal into JSON.")

	// json encoding escapes ampersand into its unicode character which is not usable directly for share
	// and fails with cloud storage. convert them back so that they are usable
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	return string(shareMessageBytes)
}

// migrateSharedURLs migrate to newest version sequentially
func migrateSharedURLs() {
	migrateSharedURLsV1ToV2()
}

func checkShareSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
	}
	if len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
	}
	if ctx.Args().Get(0) == "list" {
		if len(ctx.Args()) >= 2 {
			cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
		}
	}
	if strings.TrimSpace(ctx.Args().Get(0)) == "" {
		fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
	}
}

func setSharePalette(style string) {
	console.SetCustomPalette(map[string]*color.Color{
		"Share":   color.New(color.FgGreen, color.Bold),
		"Expires": color.New(color.FgRed, color.Bold),
		"URL":     color.New(color.FgCyan, color.Bold),
	})
	if style == "light" {
		console.SetCustomPalette(map[string]*color.Color{
			"Share":   color.New(color.FgWhite, color.Bold),
			"Expires": color.New(color.FgWhite, color.Bold),
			"URL":     color.New(color.FgWhite, color.Bold),
		})
		return
	}
	/// Add more styles here

	if style == "nocolor" {
		// All coloring options exhausted, setting nocolor safely
		console.SetNoColor()
	}
}

// mainShare - is a handler for mc share command
func mainShare(ctx *cli.Context) {
	checkShareSyntax(ctx)

	if !isSharedURLsDataDirExists() {
		shareDir, err := getSharedURLsDataDir()
		fatalIf(err.Trace(), "Unable to get shared URL data directory")

		fatalIf(createSharedURLsDataDir().Trace(), "Unable to create shared URL data directory ‘"+shareDir+"’.")
	}
	if !isSharedURLsDataFileExists() {
		shareFile, err := getSharedURLsDataFile()
		fatalIf(err.Trace(), "Unable to get shared URL data file")

		fatalIf(createSharedURLsDataFile().Trace(), "Unable to create shared URL data file ‘"+shareFile+"’.")
	}

	setSharePalette(ctx.GlobalString("colors"))

	args := ctx.Args()
	config := mustGetMcConfig()

	// if its only list, return quickly
	if args.Get(0) == "list" {
		err := doShareList()
		fatalIf(err.Trace(), "Unable to list shared URLs.")
		return
	}

	/// get first and last arguments
	url := args.Get(0) // url to be shared
	// default expiration is 7days
	expires := time.Duration(604800) * time.Second
	if len(args) == 2 {
		var err error
		expires, err = time.ParseDuration(args.Get(1))
		fatalIf(probe.NewError(err), "Unable to parse time argument.")
	}

	targetURL := getAliasURL(url, config.Aliases)

	// if recursive strip off the "..."
	err := doShareURL(stripRecursiveURL(targetURL), isURLRecursive(targetURL), expires)
	fatalIf(err.Trace(targetURL), "Unable to generate URL for sharing.")
}

// doShareList list shared url's
func doShareList() *probe.Error {
	sURLs, err := loadSharedURLsV2()
	if err != nil {
		return err.Trace()
	}
	for i, data := range sURLs.URLs {
		if time.Since(data.Date) > data.Message.Expiry {
			sURLs.URLs = append(sURLs.URLs[:i], sURLs.URLs[i+1:]...)
			continue
		}
		expiry := data.Message.Expiry - time.Since(data.Date)
		if !globalJSONFlag {
			msg := console.Colorize("Share", "Name: ")
			msg += console.Colorize("URL", data.Message.Key+"\n")
			msg += console.Colorize("Share", "Expiry: ")
			msg += console.Colorize("Expires", timeDurationToHumanizedTime(expiry))
			msg += "\n"
			console.Println(msg)
			continue
		}
		shareListBytes, err := json.Marshal(struct {
			Expiry humanizedTime `json:"expiry"`
			URL    string        `json:"url"`
			Key    string        `json:"keyName"`
		}{
			Expiry: timeDurationToHumanizedTime(expiry),
			URL:    data.Message.URL,
			Key:    data.Message.Key,
		})
		if err != nil {
			return probe.NewError(err)
		}
		console.Println(string(shareListBytes))
	}
	if err := saveSharedURLsV2(sURLs); err != nil {
		return err.Trace()
	}
	return nil
}

// doShareURL share files from target
func doShareURL(targetURL string, recursive bool, expires time.Duration) *probe.Error {
	shareDate := time.Now().UTC()
	sURLs, err := loadSharedURLsV2()
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
		sharedURL, err = newClnt.Share(expires)
		if err != nil {
			return err.Trace()
		}
		shareMessage := ShareMessage{
			Expiry: expires,
			URL:    sharedURL,
			Key:    newClnt.URL().String(),
		}
		sURLs.URLs = append(sURLs.URLs, struct {
			Date    time.Time
			Message ShareMessage
		}{
			Date:    shareDate,
			Message: shareMessage,
		})
		Prints("%s\n", shareMessage)
	}
	saveSharedURLsV2(sURLs)
	return nil
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
