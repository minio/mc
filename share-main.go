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

// shareMessage implements extended version of time.Duration, .Days() for convenience
type shareMessage struct {
	Expiry time.Duration
	URL    string
}

func (s shareMessage) Days() float64 {
	return s.Expiry.Hours() / 24
}

func (s shareMessage) Seconds() float64 {
	return s.Expiry.Seconds()
}

func (s shareMessage) Hours() float64 {
	return s.Expiry.Hours()
}

func (s shareMessage) GetDuration() time.Duration {
	return s.Expiry
}

func (s shareMessage) String() string {
	if !globalJSONFlag {
		durationString := func() string {
			if s.Expiry.Hours() > 24 {
				return fmt.Sprintf("%dd", int64(s.Days()))
			}
			return s.Expiry.String()
		}
		return console.Colorize("Share", fmt.Sprintf("Expiry: %s\n   URL: %s", durationString(), s.URL))
	}
	shareMessageBytes, err := json.Marshal(struct {
		Expires   time.Duration `json:"expireSeconds"`
		SharedURL string        `json:"sharedURL"`
	}{
		Expires:   time.Duration(s.Seconds()),
		SharedURL: s.URL,
	})
	fatalIf(probe.NewError(err), "Failed to marshal into JSON.")

	// json encoding escapes ampersand into its unicode character which is not usable directly for share
	// and fails with cloud storage. convert them back so that they are usable
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	return string(shareMessageBytes)
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

// mainShare - is a handler for mc share command
func mainShare(ctx *cli.Context) {
	checkShareSyntax(ctx)

	if !isSharedURLsDataDirExists() {
		shareDir, _ := getSharedURLsDataDir()
		fatalIf(createSharedURLsDataDir().Trace(), "Unable to create shared URL data directory ‘"+shareDir+"’.")
	}
	if !isSharedURLsDataFileExists() {
		shareFile, _ := getSharedURLsDataFile()
		fatalIf(createSharedURLsDataFile().Trace(), "Unable to create shared URL data file ‘"+shareFile+"’.")
	}

	console.SetCustomTheme(map[string]*color.Color{
		"Share":   color.New(color.FgGreen, color.Bold),
		"Expires": color.New(color.FgRed, color.Bold),
		"URL":     color.New(color.FgCyan, color.Bold),
	})

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
	err := doShareURL(stripRecursiveURL(targetURL), isURLRecursive(targetURL), shareMessage{Expiry: expires})
	fatalIf(err.Trace(targetURL), "Unable to generate URL for sharing.")
}

// doShareList list shared url's
func doShareList() *probe.Error {
	sURLs, err := loadSharedURLsV1()
	if err != nil {
		return err.Trace()
	}
	for url, data := range sURLs.URLs {
		if time.Since(data.Date) > data.Message.GetDuration() {
			delete(sURLs.URLs, url)
			continue
		}
		expiresIn := data.Message.GetDuration() - time.Since(data.Date)
		if !globalJSONFlag {
			msg := console.Colorize("Share", "Shared URL: ")
			msg += console.Colorize("URL", url+"\n")
			msg += console.Colorize("Share", "Expires-In: ")
			msg += console.Colorize("Expires", expiresIn)
			msg += "\n"
			console.Println(msg)
			continue
		}
		shareListBytes, err := json.Marshal(struct {
			ExpiresIn time.Duration
			URL       string
		}{
			ExpiresIn: time.Duration(expiresIn.Seconds()),
			URL:       url,
		})
		if err != nil {
			return probe.NewError(err)
		}
		console.Println(string(shareListBytes))
	}
	if err := saveSharedURLsV1(sURLs); err != nil {
		return err.Trace()
	}
	return nil
}

// doShareURL share files from target
func doShareURL(targetURL string, recursive bool, expires shareMessage) *probe.Error {
	shareDate := time.Now().UTC()
	sURLs, err := loadSharedURLsV1()
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
		sharedURL, err = newClnt.Share(expires.GetDuration())
		if err != nil {
			return err.Trace()
		}
		expires.URL = sharedURL
		sURLs.URLs[newClnt.URL().String()] = struct {
			Date    time.Time
			Message shareMessage
		}{
			Date:    shareDate,
			Message: expires,
		}
		console.Println(expires)
	}
	saveSharedURLsV1(sURLs)
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
