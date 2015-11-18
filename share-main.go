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
	"os"
	"path/filepath"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

var (
	shareFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of share.",
	}
)

// Share documents via URL.
var shareCmd = cli.Command{
	Name:   "share",
	Usage:  "Generate URL for sharing.",
	Action: mainShare,
	Flags:  []cli.Flag{shareFlagHelp},
	Subcommands: []cli.Command{
		shareDownload,
		shareUpload,
		shareList,
	},
	CustomHelpTemplate: `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} [FLAGS] COMMAND

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
COMMANDS:
   {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}
`,
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
		fatalIf(probe.NewError(e), "Unable to delete old ‘"+oldShareFile+"’.")
		console.Infof("Removed older version of share ‘%s’ file.\n", oldShareFile)
	}
}

// mainShare - main handler for mc share command.
func mainShare(ctx *cli.Context) {
	if ctx.Args().First() != "" { // command help.
		cli.ShowCommandHelp(ctx, ctx.Args().First())
	} else { // mc help.
		cli.ShowAppHelp(ctx)
	}

	// Sub-commands like "upload" and "download" have their own main.
}
