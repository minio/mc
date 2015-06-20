/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var sessionCmd = cli.Command{
	Name:   "session",
	Usage:  "Start sessions for cp and sync",
	Action: runSessionCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   -- TODO --

`,
}

func runSessionCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 1 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
	if !isSessionDirExists() {
		if err := createSessionDir(); err != nil {
			console.Fatalln(iodine.ToError(err))
		}
	}
	switch strings.TrimSpace(ctx.Args().First()) {
	// list resumable sessions
	case "list":
		sessionDir, err := getSessionDir()
		if err != nil {
			console.Fatalln(iodine.ToError(err))
		}
		sessions, err := ioutil.ReadDir(sessionDir)
		if err != nil {
			console.Fatalln(iodine.ToError(err))
		}
		for _, session := range sessions {
			if session.Mode().IsRegular() {
				console.Infoln(session.Name())
			}
		}
	// purge all pending sessions
	case "clear":
		sessionDir, err := getSessionDir()
		if err != nil {
			console.Fatalln(iodine.ToError(err))
		}
		sessions, err := ioutil.ReadDir(sessionDir)
		if err != nil {
			console.Fatalln(iodine.ToError(err))
		}
		for _, session := range sessions {
			if session.Mode().IsRegular() {
				err := os.Remove(filepath.Join(sessionDir, session.Name()))
				if err != nil {
					console.Fatalln(iodine.ToError(err))
				}
			}
		}
	}
}
