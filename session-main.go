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
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	sessionFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of session.",
		},
	}
)

// Manage sessions for cp and mirror.
var sessionCmd = cli.Command{
	Name:   "session",
	Usage:  "Manage saved sessions of cp and mirror operations.",
	Action: mainSession,
	Flags:  append(sessionFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] OPERATION [ARG]

OPERATION:
   resume   Resume a previously saved session.
   clear    Clear a previously saved session.
   list     List all previously saved sessions.

SESSION-ID:
   SESSION - Session can either be $SESSION-ID or "all".

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. List sessions.
      $ mc {{.Name}} list

   2. Resume session.
      $ mc {{.Name}} resume ygVIpSJs

   3. Clear session.
      $ mc {{.Name}} clear ygVIpSJs

   4. Clear session.
      $ mc {{.Name}} clear all
`,
}

// bySessionWhen is a type for sorting session metadata by time.
type bySessionWhen []*sessionV7

func (b bySessionWhen) Len() int           { return len(b) }
func (b bySessionWhen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b bySessionWhen) Less(i, j int) bool { return b[i].Header.When.Before(b[j].Header.When) }

// listSessions list all current sessions.
func listSessions() *probe.Error {
	var bySessions []*sessionV7
	for _, sid := range getSessionIDs() {
		session, err := loadSessionV7(sid)
		if err != nil {
			continue // Skip 'broken' session during listing
		}
		session.Close() // Session close right here.
		bySessions = append(bySessions, session)
	}
	// sort sessions based on time.
	sort.Sort(bySessionWhen(bySessions))
	for _, session := range bySessions {
		printMsg(session)
	}
	return nil
}

// clearSessionMessage container for clearing session messages.
type clearSessionMessage struct {
	Status    string `json:"success"`
	SessionID string `json:"sessionId"`
}

// String colorized clear session message.
func (c clearSessionMessage) String() string {
	msg := "Session ‘" + c.SessionID + "’"
	var colorizedMsg string
	switch c.Status {
	case "success":
		colorizedMsg = console.Colorize("ClearSession", msg+" cleared succesfully.")
	case "forced":
		colorizedMsg = console.Colorize("ClearSession", msg+" cleared forcefully.")
	}
	return colorizedMsg
}

// JSON jsonified clear session message.
func (c clearSessionMessage) JSON() string {
	clearSessionJSONBytes, e := json.Marshal(c)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(clearSessionJSONBytes)
}

// clearSession clear sessions.
func clearSession(sid string) {
	if sid == "all" {
		for _, sid := range getSessionIDs() {
			session, err := loadSessionV7(sid)
			fatalIf(err.Trace(sid), "Unable to load session ‘"+sid+"’.")

			fatalIf(session.Delete().Trace(sid), "Unable to load session ‘"+sid+"’.")

			printMsg(clearSessionMessage{Status: "success", SessionID: sid})
		}
		return
	}

	if !isSessionExists(sid) {
		fatalIf(errDummy().Trace(sid), "Session ‘"+sid+"’ not found.")
	}

	session, err := loadSessionV7(sid)
	if err != nil {
		// `mc session clear <broken-session-id>` assumes that user is aware that the session is unuseful
		// and wants the associated session files to be removed
		removeSessionFile(sid)
		removeSessionDataFile(sid)
		printMsg(clearSessionMessage{Status: "forced", SessionID: sid})
		return
	}

	if session != nil {
		fatalIf(session.Delete().Trace(sid), "Unable to load session ‘"+sid+"’.")
		printMsg(clearSessionMessage{Status: "success", SessionID: sid})
	}
}

func sessionExecute(s *sessionV7) {
	switch s.Header.CommandType {
	case "cp":
		doCopySession(s)
	case "mirror":
		doMirrorSession(s)
	}
}

func checkSessionSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}

	switch strings.TrimSpace(ctx.Args().First()) {
	case "list":
	case "resume":
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			fatalIf(errInvalidArgument().Trace(ctx.Args()...), "Unable to validate empty argument.")
		}
	case "clear":
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			fatalIf(errInvalidArgument().Trace(ctx.Args()...), "Unable to validate empty argument.")
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
}

// findClosestSessions to match a given string with sessions trie tree.
func findClosestSessions(session string) []string {
	sessionsTree := newTrie() // Allocate a new trie for sessions strings.
	for _, sid := range getSessionIDs() {
		sessionsTree.Insert(sid)
	}
	var closestSessions []string
	for _, value := range sessionsTree.PrefixMatch(session) {
		closestSessions = append(closestSessions, value.(string))
	}
	sort.Strings(closestSessions)
	return closestSessions
}

func mainSession(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'session' cli arguments.
	checkSessionSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Command", color.New(color.FgWhite, color.Bold))
	console.SetColor("SessionID", color.New(color.FgYellow, color.Bold))
	console.SetColor("SessionTime", color.New(color.FgGreen))
	console.SetColor("ClearSession", color.New(color.FgGreen, color.Bold))

	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session folder.")
	}

	switch strings.TrimSpace(ctx.Args().First()) {
	// list all resumable sessions.
	case "list":
		fatalIf(listSessions().Trace(ctx.Args()...), "Unable to list sessions.")
	case "resume":
		sid := strings.TrimSpace(ctx.Args().Tail().First())
		if !isSessionExists(sid) {
			closestSessions := findClosestSessions(sid)
			errorMsg := "Session ‘" + sid + "’ not found."
			if len(closestSessions) > 0 {
				errorMsg += fmt.Sprintf("\n\nDid you mean?\n")
				for _, session := range closestSessions {
					errorMsg += fmt.Sprintf("        ‘mc resume session %s’", session)
					// break on the first one, it is good enough.
					break
				}
			}
			fatalIf(errDummy().Trace(sid), errorMsg)
		}
		s, err := loadSessionV7(sid)
		fatalIf(err.Trace(sid), "Unable to load session.")

		// Restore the state of global variables from this previous session.
		s.restoreGlobals()

		savedCwd, e := os.Getwd()
		fatalIf(probe.NewError(e), "Unable to determine current working folder.")

		if s.Header.RootPath != "" {
			// change folder to RootPath.
			e = os.Chdir(s.Header.RootPath)
			fatalIf(probe.NewError(e), "Unable to change working folder to root path while resuming session.")
		}
		sessionExecute(s)
		err = s.Close()
		fatalIf(err.Trace(), "Unable to close session file properly.")

		err = s.Delete()
		fatalIf(err.Trace(), "Unable to clear session files properly.")

		// change folder back to saved path.
		e = os.Chdir(savedCwd)
		fatalIf(probe.NewError(e), "Unable to change working folder to saved path ‘"+savedCwd+"’.")
	// purge a requested pending session, if "all" purge everything.
	case "clear":
		clearSession(strings.TrimSpace(ctx.Args().Tail().First()))
	}
}
