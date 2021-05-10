// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	Name:         "list",
	Usage:        "list previously shared objects",
	Action:       mainShareList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(shareListFlags, globalFlags...),
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
