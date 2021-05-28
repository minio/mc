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
	"os"
	"path/filepath"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var (
	shareFlags = []cli.Flag{}
)

var shareSubcommands = []cli.Command{
	shareDownload,
	shareUpload,
	shareList,
}

// Share documents via URL.
var shareCmd = cli.Command{
	Name:            "share",
	Usage:           "generate URL for temporary access to an object",
	Action:          mainShare,
	Before:          setGlobalsFromContext,
	Flags:           append(shareFlags, globalFlags...),
	HideHelpCommand: true,
	Subcommands:     shareSubcommands,
}

// migrateShare migrate to newest version sequentially.
func migrateShare() {
	if !isShareDirExists() {
		return
	}

	// Shared URLs are now managed by sub-commands. So delete any old URLs file if found.
	oldShareFile := filepath.Join(mustGetShareDir(), "urls.json")
	if _, e := os.Stat(oldShareFile); e == nil {
		// Old file exits.
		e := os.Remove(oldShareFile)
		fatalIf(probe.NewError(e), "Unable to delete old `"+oldShareFile+"`.")
		console.Infof("Removed older version of share `%s` file.\n", oldShareFile)
	}
}

// mainShare - main handler for mc share command.
func mainShare(ctx *cli.Context) error {
	commandNotFound(ctx, shareSubcommands)
	return nil
	// Sub-commands like "upload" and "download" have their own main.
}
