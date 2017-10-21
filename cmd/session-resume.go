/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/trie"
)

var sessionResume = cli.Command{
	Name:   "resume",
	Usage:  "Resume a previously saved session.",
	Action: mainSessionResume,
	Flags:  globalFlags,
	Before: setGlobalsFromContext,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} SESSION-ID

SESSION-ID:
  SESSION - Session is your previously saved $SESSION-ID

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Resume session. 
     $ {{.HelpName}} ygVIpSJs
`,
}

// bySessionWhen is a type for sorting session metadata by time.
type bySessionWhen []*sessionV8

func (b bySessionWhen) Len() int           { return len(b) }
func (b bySessionWhen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b bySessionWhen) Less(i, j int) bool { return b[i].Header.When.Before(b[j].Header.When) }

// sessionExecute - run a given session.
func sessionExecute(s *sessionV8) {
	switch s.Header.CommandType {
	case "cp":
		doCopySession(s)
	}
}

// findClosestSessions to match a given string with sessions trie tree.
func findClosestSessions(session string) []string {
	sessionsTree := trie.NewTrie() // Allocate a new trie for sessions strings.
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

// checkSessionResumeSyntax - Validate session resume command.
func checkSessionResumeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "resume", 1) // last argument is exit code
	}
}

// mainSessionResume - Main session resume function.
func mainSessionResume(ctx *cli.Context) error {
	// Validate session resume syntax.
	checkSessionResumeSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("Command", color.New(color.FgWhite, color.Bold))
	console.SetColor("SessionID", color.New(color.FgYellow, color.Bold))
	console.SetColor("SessionTime", color.New(color.FgGreen))

	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session folder.")
	}

	sessionID := ctx.Args().Get(0)
	if !isSessionExists(sessionID) {
		closestSessions := findClosestSessions(sessionID)
		errorMsg := "Session `" + sessionID + "` not found."
		if len(closestSessions) > 0 {
			errorMsg += fmt.Sprintf("\n\nDid you mean?\n")
			for _, session := range closestSessions {
				errorMsg += fmt.Sprintf("        `mc resume session %s`", session)
				// break on the first one, it is good enough.
				break
			}
		}
		fatalIf(errDummy().Trace(sessionID), errorMsg)
	}
	resumeSession(sessionID)
	return nil
}

// resumeSession - Resumes a session specified by sessionID.
func resumeSession(sessionID string) {
	s, err := loadSessionV8(sessionID)
	fatalIf(err.Trace(sessionID), "Unable to load session.")
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
	fatalIf(probe.NewError(e), "Unable to change working folder to saved path `"+savedCwd+"`.")

}
