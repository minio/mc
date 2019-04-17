/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var sessionClearFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "force",
		Usage: "force a dangerous clear operation.",
	},
}

var sessionClear = cli.Command{
	Name:            "clear",
	Usage:           "clear interrupted session",
	Action:          mainSessionClear,
	Before:          setGlobalsFromContext,
	Flags:           append(sessionClearFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} SESSION-ID|all

SESSION-ID:
  SESSION - Session can either be $SESSION-ID or all

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Clear session.
     $ {{.HelpName}} ygVIpSJs
  2. Clear all sessions.
     $ {{.HelpName}} all
  3. Forcefully clear an obsolete session.
     $ {{.HelpName}} ygVIpSJs --force
`,
}

// clearSessionMessage container for clearing session messages.
type clearSessionMessage struct {
	Status    string `json:"success"`
	SessionID string `json:"sessionId"`
}

// String colorized clear session message.
func (c clearSessionMessage) String() string {
	msg := "Session `" + c.SessionID + "`"
	var colorizedMsg string
	switch c.Status {
	case "success":
		colorizedMsg = console.Colorize("ClearSession", msg+" cleared successfully.")
	case "forced":
		colorizedMsg = console.Colorize("ClearSession", msg+" cleared forcefully.")
	}
	return colorizedMsg
}

// JSON jsonified clear session message.
func (c clearSessionMessage) JSON() string {
	clearSessionJSONBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(clearSessionJSONBytes)
}

// forceClear - Remove a saved session.
// Used if --force flag is applied.
func forceClear(sid string, session *sessionV8) {
	if session != nil {
		if err := session.Delete().Trace(sid); err == nil {
			// Force unnecesseray removal successful.
			printMsg(clearSessionMessage{Status: "success", SessionID: sid})
			return
		}
	}
	// Remove obsolete session files.
	removeSessionFile(sid)
	removeSessionDataFile(sid)
	printMsg(clearSessionMessage{Status: "forced", SessionID: sid})
}

// clearSession clear sessions.
func clearSession(sid string, isForce bool) {
	var toRemove []string
	if sid == "all" {
		toRemove = getSessionIDs()
	} else {
		toRemove = append(toRemove, sid)
	}
	for _, sid := range toRemove {
		session, err := loadSessionV8(sid)
		if !isForce {
			fatalIf(err.Trace(sid), "Unable to load session `"+sid+"`. Use --force flag to remove obsolete session files.")

			fatalIf(session.Delete().Trace(sid), "Unable to remove session `"+sid+"`. Use --force flag to remove obsolete session files.")

			printMsg(clearSessionMessage{Status: "success", SessionID: sid})
			continue
		}
		// Forced removal of a session.
		forceClear(sid, session)
	}
	return
}

// checkSessionClearSyntax - Check syntax of 'session clear sid'.
func checkSessionClearSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "clear", 1) // last argument is exit code
	}
}

// mainSessionClear - Main session clear.
func mainSessionClear(ctx *cli.Context) error {
	// Check command arguments
	checkSessionClearSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("Command", color.New(color.FgWhite, color.Bold))
	console.SetColor("SessionID", color.New(color.FgYellow, color.Bold))
	console.SetColor("SessionTime", color.New(color.FgGreen))
	console.SetColor("ClearSession", color.New(color.FgGreen, color.Bold))

	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session folder.")
	}
	// Set command flags.
	isForce := ctx.Bool("force")

	// Retrieve requested session id.
	sessionID := ctx.Args().Get(0)

	// Purge requested session id or all pending sessions.
	clearSession(sessionID, isForce)
	return nil
}
