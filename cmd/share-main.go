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
	"os"
	"path/filepath"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	shareFlags = []cli.Flag{}
)

// Share documents via URL.
var shareCmd = cli.Command{
	Name:            "share",
	Usage:           "generate URL for temporary access to an object",
	Action:          mainShare,
	Before:          setGlobalsFromContext,
	Flags:           append(shareFlags, globalFlags...),
	HideHelpCommand: true,
	Subcommands: []cli.Command{
		shareDownload,
		shareUpload,
		shareList,
	},
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
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "upload" and "download" have their own main.
}
