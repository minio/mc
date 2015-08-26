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
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Manage sessions for cp and mirror.
var sessionCmd = cli.Command{
	Name:   "session",
	Usage:  "Manage sessions for cp and mirror.",
	Action: mainSession,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} [SESSION] {{if .Description}}

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
      $ mc {{.Name}} clear [SESSION]|[all]
`,
}

// bySessionWhen is a type for sorting session metadata by time
type bySessionWhen []*sessionV2

func (b bySessionWhen) Len() int           { return len(b) }
func (b bySessionWhen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b bySessionWhen) Less(i, j int) bool { return b[i].Header.When.Before(b[j].Header.When) }

func listSessions() *probe.Error {
	var bySessions []*sessionV2
	for _, sid := range getSessionIDs() {
		s, err := loadSessionV2(sid)
		if err != nil {
			return err.Trace()
		}
		bySessions = append(bySessions, s)
	}
	// sort sessions based on time
	sort.Sort(bySessionWhen(bySessions))
	for _, session := range bySessions {
		console.Println(session)
	}
	return nil
}

// ClearSessionMessage container for clearing session messages
type ClearSessionMessage struct {
	Status    string `json:"success"`
	SessionID string `json:"session-id"`
}

func (c ClearSessionMessage) String() string {
	if !globalJSONFlag {
		return console.Colorize("ClearSession", "Session ‘"+c.SessionID+"’ cleared successfully")
	}
	clearSessionJSONBytes, err := json.Marshal(c)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(clearSessionJSONBytes)
}

func clearSession(sid string) {
	if sid == "all" {
		for _, sid := range getSessionIDs() {
			session, err := loadSessionV2(sid)
			fatalIf(err.Trace(sid), "Unable to load session ‘"+sid+"’.")

			fatalIf(session.Delete().Trace(sid), "Unable to load session ‘"+sid+"’.")
			console.Println(ClearSessionMessage{
				Status:    "success",
				SessionID: sid,
			})
		}
		return
	}

	if !isSession(sid) {
		fatalIf(errDummy().Trace(), "Session ‘"+sid+"’ not found.")
	}

	session, err := loadSessionV2(sid)
	fatalIf(err.Trace(sid), "Unable to load session ‘"+sid+"’.")

	if session != nil {
		fatalIf(session.Delete().Trace(sid), "Unable to load session ‘"+sid+"’.")
		console.Println(ClearSessionMessage{
			Status:    "success",
			SessionID: sid,
		})
	}
}

func sessionExecute(s *sessionV2) {
	switch s.Header.CommandType {
	case "cp":
		doCopyCmdSession(s)
	case "mirror":
		doMirrorCmdSession(s)
	}
}

func checkSessionSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 1 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}

	switch strings.TrimSpace(ctx.Args().First()) {
	case "list":
	case "resume":
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
		}
	case "clear":
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
}

func mainSession(ctx *cli.Context) {
	checkSessionSyntax(ctx)

	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session directory.")
	}

	console.SetCustomTheme(map[string]*color.Color{
		"Command":      color.New(color.FgWhite, color.Bold),
		"SessionID":    color.New(color.FgYellow, color.Bold),
		"SessionTime":  color.New(color.FgGreen),
		"ClearSession": color.New(color.FgGreen, color.Bold),
	})

	switch strings.TrimSpace(ctx.Args().First()) {
	// list resumable sessions
	case "list":
		fatalIf(listSessions().Trace(), "Unable to list sessions.")
	case "resume":
		sid := strings.TrimSpace(ctx.Args().Tail().First())
		if !isSession(sid) {
			fatalIf(errDummy().Trace(), "Session ‘"+sid+"’ not found.")
		}

		s, err := loadSessionV2(sid)
		fatalIf(err.Trace(sid), "Unable to load session.")

		// extra check for testing purposes
		if s == nil {
			return
		}

		savedCwd, e := os.Getwd()
		fatalIf(probe.NewError(e), "Unable to verify your current working folder.")

		if s.Header.RootPath != "" {
			// chdir to RootPath
			e = os.Chdir(s.Header.RootPath)
			fatalIf(probe.NewError(e), "Unable to change directory to root path while resuming session.")
		}
		sessionExecute(s)
		err = s.Close()
		fatalIf(err.Trace(), "Unable to close session file properly.")

		err = s.Delete()
		fatalIf(err.Trace(), "Unable to clear session files properly.")

		// chdir back to saved path
		e = os.Chdir(savedCwd)
		fatalIf(probe.NewError(e), "Unable to change directory to saved path ‘"+savedCwd+"’.")

	// purge a requested pending session, if "all" purge everything
	case "clear":
		clearSession(strings.TrimSpace(ctx.Args().Tail().First()))
	}
}
