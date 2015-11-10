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
	"encoding/json"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// make a bucket or folder.
var mbCmd = cli.Command{
	Name:   "mb",
	Usage:  "Make a bucket or folder.",
	Action: mainMakeBucket,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET ...]

EXAMPLES:
   1. Create a bucket on Amazon S3 cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/public-document-store

   2. Make a folder on local filesystem with space characters
      $ mc {{.Name}} 'My Documents'

   3. Create a bucket on Minio cloud storage.
      $ mc {{.Name}} https://play.minio.io:9000/mongodb-backup

   4. Create a bucket on Google Cloud Storage.
      $ mc {{.Name}} https://storage.googleapis.com/miniocloud
`,
}

// makeBucketMessage is container for make bucket success and failure messages
type makeBucketMessage struct {
	Status string `json:"status"`
	Bucket string `json:"bucket"`
}

// String colorized make bucket message
func (s makeBucketMessage) String() string {
	return console.Colorize("MakeBucket", "Bucket created successfully  ‘"+s.Bucket+"’")
}

// JSON jsonified make bucket message
func (s makeBucketMessage) JSON() string {
	makeBucketJSONBytes, err := json.Marshal(s)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(makeBucketJSONBytes)
}

func checkMakeBucketSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "mb", 1) // last argument is exit code
	}
	for _, arg := range ctx.Args() {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
		}
	}
}

func setMakeBucketPalette(style string) {
	console.SetCustomPalette(map[string]*color.Color{
		"MakeBucket": color.New(color.FgGreen, color.Bold),
	})
	if style == "light" {
		console.SetCustomPalette(map[string]*color.Color{
			"MakeBucket": color.New(color.FgWhite, color.Bold),
		})
		return
	}
	if style == "nocolor" {
		// All coloring options exhausted, setting nocolor safely
		console.SetNoColor()
	}
}

// mainMakeBucket is the handler for mc mb command
func mainMakeBucket(ctx *cli.Context) {
	checkMakeBucketSyntax(ctx)

	setMakeBucketPalette(ctx.GlobalString("colors"))

	config := mustGetMcConfig()
	for _, arg := range ctx.Args() {
		targetURL := getAliasURL(arg, config.Aliases)

		// Instantiate client for URL.
		clnt, err := url2Client(targetURL)
		fatalIf(err.Trace(targetURL), "Invalid target target ‘"+targetURL+"’.")

		// Make bucket.
		fatalIf(clnt.MakeBucket().Trace(), "Unable to make bucket ‘"+targetURL+"’.")

		// Successfully created a bucket.
		printMsg(makeBucketMessage{Status: "success", Bucket: targetURL})
	}
}
