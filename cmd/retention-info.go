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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
)

var (
	retentionInfoFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "show retention info recursively",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "show retention info of specific object version",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "roll back object(s) to current version at specified time",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "show retention info on object(s) and all its versions",
		},
		cli.BoolFlag{
			Name:  "default",
			Usage: "show bucket default retention mode",
		},
	}
)

var retentionInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "show retention for object(s)",
	Action:       mainRetentionInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(retentionInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [governance | compliance] VALIDITY TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Show object retention for a specific object
     $ {{.HelpName}} myminio/mybucket/prefix/obj.csv

  2. Show object retention for recursively for all objects at a given prefix
     $ {{.HelpName}} myminio/mybucket/prefix --recursive

  3. Show object retention to a specific version of a specific object
     $ {{.HelpName}} myminio/mybucket/prefix/obj.csv --version-id "3Jr2x6fqlBUsVzbvPihBO3HgNpgZgAnp"

  4. Show object retention for recursively for all versions of all objects under prefix
     $ {{.HelpName}} myminio/mybucket/prefix --recursive --versions

  5. Show default lock retention configuration for a bucket
     $ {{.HelpName}} --default myminio/mybucket/
`}

func parseInfoRetentionArgs(cliCtx *cli.Context) (target, versionID string, recursive bool, timeRef time.Time, withVersions, defaultMode bool) {
	args := cliCtx.Args()

	target = args[0]
	if target == "" {
		fatalIf(errInvalidArgument().Trace(), "invalid target url '%v'", target)
	}

	versionID = cliCtx.String("version-id")
	timeRef = parseRewindFlag(cliCtx.String("rewind"))
	withVersions = cliCtx.Bool("versions")
	recursive = cliCtx.Bool("recursive")
	defaultMode = cliCtx.Bool("default")
	return
}

// Structured message depending on the type of console.
type retentionInfoMessage struct {
	Mode      minio.RetentionMode `json:"mode"`
	Until     time.Time           `json:"until"`
	URLPath   string              `json:"urlpath"`
	VersionID string              `json:"versionID"`
	Status    string              `json:"status"`
	Err       error               `json:"error"`
}

type retentionInfoMessageList retentionInfoMessage

func (m *retentionInfoMessageList) SetErr(e error) {
	m.Err = e
}

func (m *retentionInfoMessageList) SetStatus(status string) {
	m.Status = status
}

func (m *retentionInfoMessageList) SetMode(mode minio.RetentionMode) {
	m.Mode = mode
}

func (m *retentionInfoMessageList) SetUntil(until time.Time) {
	m.Until = until
}

// Colorized message for console printing.
func (m retentionInfoMessageList) String() string {
	if m.Err != nil {
		return console.Colorize("RetentionFailure", fmt.Sprintf("Unable to get get object retention on `%s`: %s", m.URLPath, m.Err))
	}

	var msg string
	var retentionField string

	if m.Mode == "" {
		retentionField += console.Colorize("RetentionNotFound", "NO RETENTION")
	} else {
		exp := ""
		if m.Mode == minio.Governance {
			now := time.Now()
			if now.After(m.Until) {
				exp = "EXPIRED"
			}
		}
		retentionField += console.Colorize("RetentionSuccess", m.Mode.String()) + " " + console.Colorize("RetentionExpired", exp)
	}

	msg += "[ " + centerText(retentionField, 18) + " ]  "

	if m.VersionID != "" {
		msg += console.Colorize("RetentionVersionID", m.VersionID+"  ")
	}

	msg += m.URLPath
	return msg
}

// JSON'ified message for scripting.
func (m retentionInfoMessageList) JSON() string {
	if m.Err != nil {
		m.Status = "failure"
	}
	msgBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

type retentionInfoMessageRecord retentionInfoMessage

func (m *retentionInfoMessageRecord) SetErr(e error) {
	m.Err = e
}

func (m *retentionInfoMessageRecord) SetStatus(status string) {
	m.Status = status
}

func (m *retentionInfoMessageRecord) SetMode(mode minio.RetentionMode) {
	m.Mode = mode
}

func (m *retentionInfoMessageRecord) SetUntil(until time.Time) {
	m.Until = until
}

// Colorized message for console printing.
func (m retentionInfoMessageRecord) String() string {
	if m.Err != nil {
		return console.Colorize("RetentionFailure", fmt.Sprintf("Unable to get object retention on `%s`: %s", m.URLPath, m.Err))
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "Name    : %s\n", console.Colorize("RetentionSuccess", m.URLPath))

	if m.VersionID != "" {
		fmt.Fprintf(&msg, "Version : %s\n", console.Colorize("RetentionSuccess", m.VersionID))
	}

	fmt.Fprintf(&msg, "Mode    : ")
	if m.Mode == "" {
		fmt.Fprintf(&msg, console.Colorize("RetentionNotFound", "NO RETENTION"))
	} else {
		fmt.Fprintf(&msg, console.Colorize("RetentionSuccess", m.Mode))
		if !m.Until.IsZero() {
			msg.WriteString(", ")
			exp := ""
			now := time.Now()
			if now.After(m.Until) {
				prettyDuration := timeDurationToHumanizedDuration(now.Sub(m.Until)).StringShort()
				exp = console.Colorize("RetentionExpired", "expired "+prettyDuration+" ago")
			} else {
				prettyDuration := timeDurationToHumanizedDuration(m.Until.Sub(now)).StringShort()
				exp = console.Colorize("RetentionSuccess", "expiring in "+prettyDuration)
			}
			fmt.Fprintf(&msg, exp)
		}
	}
	fmt.Fprintf(&msg, "\n")
	return msg.String()
}

// JSON'ified message for scripting.
func (m retentionInfoMessageRecord) JSON() string {
	if m.Err != nil {
		m.Status = "failure"
	}
	msgBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

type retentionInfoMsg interface {
	message
	SetErr(error)
	SetStatus(string)
	SetMode(minio.RetentionMode)
	SetUntil(time.Time)
}

// Show retention info for a single object or version
func infoRetentionSingle(ctx context.Context, alias, url, versionID string, listStyle bool) *probe.Error {
	newClnt, err := newClientFromAlias(alias, url)
	if err != nil {
		return err
	}

	var msg retentionInfoMsg

	if listStyle {
		msg = &retentionInfoMessageList{
			URLPath:   urlJoinPath(alias, url),
			VersionID: versionID,
		}
	} else {
		msg = &retentionInfoMessageRecord{
			URLPath:   urlJoinPath(alias, url),
			VersionID: versionID,
		}
	}

	mode, until, err := newClnt.GetObjectRetention(ctx, versionID)
	if err != nil {
		errResp := minio.ToErrorResponse(err.ToGoError())
		if errResp.Code != "NoSuchObjectLockConfiguration" {
			msg.SetErr(err.ToGoError())
			msg.SetStatus("failure")
			printMsg(msg)
			return err
		}
		err = nil
	}

	msg.SetStatus("success")
	msg.SetMode(mode)
	msg.SetUntil(until)

	printMsg(msg)
	return err
}

// Get Retention for one object/version or many objects within a given prefix.
func getRetention(ctx context.Context, target, versionID string, timeRef time.Time, withOlderVersions, isRecursive bool) error {
	clnt, err := newClient(target)
	if err != nil {
		fatalIf(err.Trace(), "Unable to parse the provided url.")
	}

	// Quit early if urlStr does not point to an S3 server
	switch clnt.(type) {
	case *S3Client:
	default:
		fatal(errDummy().Trace(), "Retention is supported only for S3 servers.")
	}

	alias, urlStr, _ := mustExpandAlias(target)
	if versionID != "" || !isRecursive && !withOlderVersions {
		err := infoRetentionSingle(ctx, alias, urlStr, versionID, false)
		if err != nil {
			return exitStatus(globalErrorExitStatus)
		}
		return nil
	}

	lstOptions := ListOptions{Recursive: isRecursive, ShowDir: DirNone}
	if !timeRef.IsZero() {
		lstOptions.WithOlderVersions = withOlderVersions
		lstOptions.WithDeleteMarkers = true
		lstOptions.TimeRef = timeRef
	}

	var cErr error
	var atLeastOneObjectOrVersionFound bool

	for content := range clnt.List(ctx, lstOptions) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}
		// The spec does not allow setting retention on delete marker
		if content.IsDeleteMarker {
			continue
		}

		if !isRecursive && alias+getKey(content) != getStandardizedURL(target) {
			break
		}

		err := infoRetentionSingle(ctx, alias, content.URL.String(), content.VersionID, true)
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Invalid URL")
			cErr = exitStatus(globalErrorExitStatus)
			continue
		}

		atLeastOneObjectOrVersionFound = true
	}

	if !atLeastOneObjectOrVersionFound {
		errorIf(errDummy().Trace(clnt.GetURL().String()), "Unable to find any object/version to show its retention.")
		cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
	}

	return cErr
}

// main for retention info command.
func mainRetentionInfo(cliCtx *cli.Context) error {
	ctx, cancelSetRetention := context.WithCancel(globalContext)
	defer cancelSetRetention()

	console.SetColor("RetentionSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("RetentionNotFound", color.New(color.FgYellow))
	console.SetColor("RetentionVersionID", color.New(color.FgGreen))
	console.SetColor("RetentionExpired", color.New(color.FgRed, color.Bold))
	console.SetColor("RetentionFailure", color.New(color.FgYellow))

	if len(cliCtx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(cliCtx, "info", 1)
	}

	target, versionID, recursive, rewind, withVersions, bucketMode := parseInfoRetentionArgs(cliCtx)

	checkObjectLockSupport(ctx, target)

	if bucketMode {
		return showBucketLock(target)
	}

	if withVersions && rewind.IsZero() {
		rewind = time.Now().UTC()
	}

	return getRetention(ctx, target, versionID, rewind, withVersions, recursive)
}
