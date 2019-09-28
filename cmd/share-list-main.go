/*
 * MinIO Client (C) 2014, 2015 MinIO, Inc.
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

package cmd

import (
	"fmt"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	shareListFlags = []cli.Flag{}
)

// Share documents via URL.
var shareList = cli.Command{
	Name:   "list",
	Usage:  "list previously shared objects",
	Action: mainShareList,
	Before: setGlobalsFromContext,
	Flags:  append(shareListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} COMMAND - {{.Usage}}

USAGE:
  {{.HelpName}} COMMAND

COMMAND:
  upload:   list previously shared access to uploads.
  download: list previously shared access to downloads.

EXAMPLES:
  1. List previously shared downloads, that haven't expired yet.
      {{.Prompt}} {{.HelpName}} download

  2. List previously shared uploads, that haven't expired yet.
      {{.Prompt}} {{.HelpName}} upload
`,
}

// validate command-line args.
func checkShareListSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !args.Present() || (args.First() != "upload" && args.First() != "download") {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code.
	}
}

// doShareList list shared url's.
func doShareList(cmd string) *probe.Error {
	if cmd != "upload" && cmd != "download" {
		return probe.NewError(fmt.Errorf("Unknown argument `%s` passed", cmd))
	}

	// Fetch defaults.
	uploadsFile := getShareUploadsFile()
	downloadsFile := getShareDownloadsFile()

	// Load previously saved upload-shares.
	shareDB := newShareDBV1()

	// if upload - read uploads file.
	if cmd == "upload" {
		if err := shareDB.Load(uploadsFile); err != nil {
			return err.Trace(uploadsFile)
		}
	}

	// if download - read downloads file.
	if cmd == "download" {
		if err := shareDB.Load(downloadsFile); err != nil {
			return err.Trace(downloadsFile)
		}
	}

	// Print previously shared entries.
	for shareURL, share := range shareDB.Shares {
		printMsg(shareMesssage{
			ObjectURL:   share.URL,
			ShareURL:    shareURL,
			TimeLeft:    share.Expiry - time.Since(share.Date),
			ContentType: share.ContentType,
		})
	}
	return nil
}

// main entry point for share list.
func mainShareList(ctx *cli.Context) error {

	// validate command-line args.
	checkShareListSyntax(ctx)

	// Additional command speific theme customization.
	shareSetColor()

	// Initialize share config folder.
	initShareConfig()

	// List shares.
	fatalIf(doShareList(ctx.Args().First()).Trace(), "Unable to list previously shared URLs.")
	return nil
}
