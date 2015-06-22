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
   1. List sessions
      $ mc {{.Name}} list

   2. Resume session
      $ mc {{.Name}} resume [SESSION]

   3. Clear session
      $ mc {{.Name}} clear [SESSION]|[*]

`,
}

func listSessions(sdir string) {
	sdir, err := getSessionDir()
	if err != nil {
		console.Fatalln(iodine.ToError(err))
	}
	sessions, err := ioutil.ReadDir(sdir)
	if err != nil {
		console.Fatalln(iodine.ToError(err))
	}
	for _, session := range sessions {
		if session.Mode().IsRegular() {
			console.Infoln(session.Name())
		}
	}
}

func resumeSession(sid string) (*sessionV1, error) {
	sfile, err := getSessionFile(sid)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	_, err = os.Stat(sfile)
	if err != nil {
		return nil, iodine.New(errInvalidSessionID{id: sid}, nil)
	}
	s, err := loadSession(sid)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return s, nil
}

func clearSession(sid string) error {
	if sid == "*" {
		sdir, err := getSessionDir()
		if err != nil {
			return iodine.New(err, nil)
		}
		sessions, err := ioutil.ReadDir(sdir)
		if err != nil {
			return iodine.New(err, nil)
		}
		for _, session := range sessions {
			if session.Mode().IsRegular() {
				err := os.Remove(filepath.Join(sdir, session.Name()))
				if err != nil {
					return iodine.New(err, nil)
				}
			}
		}
		return nil
	}
	sfile, err := getSessionFile(sid)
	if err != nil {
		return iodine.New(err, nil)
	}
	err = os.Remove(sfile)
	if err != nil {
		return iodine.New(errInvalidSessionID{id: sid}, nil)
	}
	return nil
}

func sessionExecute(bar barSend, s *sessionV1) {
	switch s.CommandType {
	case "cp":
		for cps := range doCopyCmdSession(bar, s) {
			if cps.Error != nil {
				console.Errors(ErrorMessage{
					Message: "Failed with",
					Error:   iodine.New(cps.Error, nil),
				})
			}
			if cps.Done {
				if err := saveSession(s); err != nil {
					console.Fatalln(iodine.ToError(err))
				}
				os.Exit(0)
			}
		}
	case "sync":
		for ss := range doSyncCmdSession(bar, s) {
			if ss.Error != nil {
				console.Errors(ErrorMessage{
					Message: "Failed with",
					Error:   iodine.New(ss.Error, nil),
				})
			}
			if ss.Done {
				if err := saveSession(s); err != nil {
					console.Fatalln(iodine.ToError(err))
				}
				// this os.Exit is needed really to exit in-case of "os.Interrupt"
				os.Exit(0)
			}
		}
	}
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
		listSessions(sessionDir)
	case "resume":
		if len(ctx.Args().Tail()) != 1 {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		sid := strings.TrimSpace(ctx.Args().Tail().First())
		s, err := resumeSession(sid)
		if err != nil {
			console.Fatalln(iodine.ToError(err))
		}
		var bar barSend
		// set up progress bar
		if !globalQuietFlag {
			bar = newCpBar()
		}
		sessionExecute(bar, s)
		if !globalQuietFlag {
			bar.Finish()
			if err := clearSession(sid); err != nil {
				console.Fatalln(iodine.ToError(err))
			}
		}
	// purge a requested pending session, if "*" purge everything
	case "clear":
		if len(ctx.Args().Tail()) != 1 {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		if err := clearSession(strings.TrimSpace(ctx.Args().Tail().First())); err != nil {
			console.Fatalln(iodine.ToError(err))
		}
	}
}
