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
	"sort"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var sessionList = cli.Command{
	Name:   "list",
	Usage:  "List all previously saved sessions.",
	Before: setGlobalsFromContext,
	Action: mainSessionList,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}}

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List sessions.
     $ {{.HelpName}}
`,
}

// listSessions list all current sessions.
func listSessions() *probe.Error {
	var bySessions []*sessionV8
	for _, sid := range getSessionIDs() {
		session, err := loadSessionV8(sid)
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

func checkSessionListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 0 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

func mainSessionList(ctx *cli.Context) error {
	// Check 'session list'.
	checkSessionListSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("Command", color.New(color.FgWhite, color.Bold))
	console.SetColor("SessionID", color.New(color.FgYellow, color.Bold))
	console.SetColor("SessionTime", color.New(color.FgGreen))

	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session folder.")
	}
	// List all resumable sessions.
	fatalIf(listSessions().Trace(ctx.Args()...), "Unable to list sessions.")
	return nil
}
