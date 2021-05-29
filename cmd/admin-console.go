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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

const logTimeFormat string = "15:04:05 MST 01/02/2006"

var adminConsoleFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "limit, l",
		Usage: "show last n log entries",
		Value: 10,
	},
	cli.StringFlag{
		Name:  "type, t",
		Usage: "list error logs by type. Valid options are '[minio, application, all]'",
		Value: "all",
	},
}

var adminConsoleCmd = cli.Command{
	Name:            "console",
	Usage:           "show console logs for MinIO server",
	Action:          mainAdminConsole,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminConsoleFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [NODENAME]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show console logs for a MinIO server with alias 'play'
     {{.Prompt}} {{.HelpName}} play

  2. Show last 5 log entries for node 'node1' on MinIO server with alias 'cluster1'
     {{.Prompt}} {{.HelpName}} --limit 5 cluster1 node1

  3. Show application error logs on MinIO server with alias 'play'
     {{.Prompt}} {{.HelpName}} --type application play
`,
}

func checkAdminLogSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 3 {
		cli.ShowCommandHelpAndExit(ctx, "console", 1) // last argument is exit code
	}
}

// Extend madmin.LogInfo to add String() and JSON() methods
type logMessage struct {
	Status string `json:"status"`
	madmin.LogInfo
}

// JSON - jsonify loginfo
func (l logMessage) JSON() string {
	l.Status = "success"
	logJSON, err := json.MarshalIndent(&l, "", " ")
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(logJSON)

}
func getLogTime(lt string) string {
	tm, err := time.Parse(time.RFC3339Nano, lt)
	if err != nil {
		return lt
	}
	return tm.Format(logTimeFormat)
}

// String - return colorized loginfo as string.
func (l logMessage) String() string {
	var hostStr string
	var b = &strings.Builder{}
	if l.NodeName != "" {
		hostStr = fmt.Sprintf("%s ", colorizedNodeName(l.NodeName))
	}
	log := l.LogInfo
	if log.ConsoleMsg != "" {
		if strings.HasPrefix(log.ConsoleMsg, "\n") {
			fmt.Fprintf(b, "%s\n", hostStr)
			log.ConsoleMsg = strings.TrimPrefix(log.ConsoleMsg, "\n")
		}
		fmt.Fprintf(b, "%s %s", hostStr, log.ConsoleMsg)
		return b.String()
	}
	if l.API != nil {
		apiString := "API: " + l.API.Name + "("
		if l.API.Args != nil && l.API.Args.Bucket != "" {
			apiString = apiString + "bucket=" + l.API.Args.Bucket
		}
		if l.API.Args != nil && l.API.Args.Object != "" {
			apiString = apiString + ", object=" + l.API.Args.Object
		}
		apiString += ")"
		fmt.Fprintf(b, "\n%s %s", hostStr, console.Colorize("API", apiString))
	}
	if l.Time != "" {
		fmt.Fprintf(b, "\n%s Time: %s", hostStr, getLogTime(l.Time))
	}
	if l.DeploymentID != "" {
		fmt.Fprintf(b, "\n%s DeploymentID: %s", hostStr, l.DeploymentID)
	}
	if l.RequestID != "" {
		fmt.Fprintf(b, "\n%s RequestID: %s", hostStr, l.RequestID)
	}
	if l.RemoteHost != "" {
		fmt.Fprintf(b, "\n%s RemoteHost: %s", hostStr, l.RemoteHost)
	}
	if l.UserAgent != "" {
		fmt.Fprintf(b, "\n%s UserAgent: %s", hostStr, l.UserAgent)
	}
	if l.Trace != nil {
		if l.Trace.Message != "" {
			fmt.Fprintf(b, "\n%s Error: %s", hostStr, console.Colorize("LogMessage", l.Trace.Message))
		}
		if l.Trace.Variables != nil {
			for key, value := range l.Trace.Variables {
				if value != "" {
					fmt.Fprintf(b, "\n%s %s=%s", hostStr, key, value)
				}
			}
		}
		if l.Trace.Source != nil {
			traceLength := len(l.Trace.Source)
			for i, element := range l.Trace.Source {
				fmt.Fprintf(b, "\n%s %8v: %s", hostStr, traceLength-i, element)
			}
		}
	}
	logMsg := strings.TrimPrefix(b.String(), "\n")
	return fmt.Sprintf("%s\n", logMsg)
}

// mainAdminConsole - the entry function of console command
func mainAdminConsole(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminLogSyntax(ctx)
	console.SetColor("LogMessage", color.New(color.Bold, color.FgRed))
	console.SetColor("Api", color.New(color.Bold, color.FgWhite))
	for _, c := range colors {
		console.SetColor(fmt.Sprintf("Node%d", c), color.New(c))
	}
	aliasedURL := ctx.Args().Get(0)
	var node string
	if len(ctx.Args()) > 1 {
		node = ctx.Args().Get(1)
	}
	var limit int
	if ctx.IsSet("limit") {
		limit = ctx.Int("limit")
		if limit <= 0 {
			fatalIf(errInvalidArgument().Trace(ctx.Args()...), "please set a proper limit, for example: '--limit 5' to display last 5 logs, omit this flag to display all available logs")
		}
	}
	logType := strings.ToLower(ctx.String("type"))
	if logType != "minio" && logType != "application" && logType != "all" {
		fatalIf(errInvalidArgument().Trace(ctx.Args()...), "Invalid value for --type flag. Valid options are [minio, application, all]")
	}
	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	// Start listening on all console log activity.
	logCh := client.GetLogs(ctxt, node, limit, logType)
	for logInfo := range logCh {
		if logInfo.Err != nil {
			fatalIf(probe.NewError(logInfo.Err), "Unable to listen to console logs")
		}
		// drop nodeName from output if specified as cli arg
		if node != "" {
			logInfo.NodeName = ""
		}
		printMsg(logMessage{LogInfo: logInfo})
	}
	return nil
}
