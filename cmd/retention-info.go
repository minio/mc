/*
 * MinIO Client (C) 2019-2020 MinIO, Inc.
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
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
)

var (
	retentionInfoFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply retention recursively",
		},
		cli.StringFlag{
			Name:  "version-id",
			Usage: "apply legal hold to a specific object version",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "Move back in time",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "Pick earlier versions",
		},
	}
)

var retentionInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "show retention for object(s)",
	Action: mainRetentionInfo,
	Before: setGlobalsFromContext,
	Flags:  append(retentionInfoFlags, globalFlags...),
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

`}

func parseInfoRetentionArgs(cliCtx *cli.Context) (target, versionID string, recursive bool, timeRef time.Time, withVersions bool) {
	args := cliCtx.Args()

	target = args[0]
	if target == "" {
		fatalIf(errInvalidArgument().Trace(), "invalid target url '%v'", target)
	}

	versionID = cliCtx.String("version-id")
	timeRef = parseRewindFlag(cliCtx.String("rewind"))
	withVersions = cliCtx.Bool("versions")
	recursive = cliCtx.Bool("recursive")
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

// Colorized message for console printing.
func (m retentionInfoMessage) String() string {
	if m.Err != nil {
		return console.Colorize("RetentionFailure", fmt.Sprintf("Cannot get object retention on `%s`: %s", m.URLPath, m.Err))
	}

	var msg string
	msg += fmt.Sprintf("%s", m.URLPath)
	if m.VersionID != "" {
		msg += fmt.Sprintf(", Version ID: %s", m.VersionID)
	}

	if m.Mode == "" {
		msg += ", No retention found"
		return console.Colorize("RetentionSuccess", msg)
	}

	msg += fmt.Sprintf(", Mode: %s", m.Mode)
	now := time.Now()
	if now.Before(m.Until) {
		msg += fmt.Sprintf(", Expiring in %s", m.Until.Sub(now))
	} else {
		msg += ", " + console.Colorize("RetentionExpired", fmt.Sprintf("Expired %s ago", now.Sub(m.Until)))
	}

	return console.Colorize("RetentionSuccess", msg)
}

// JSON'ified message for scripting.
func (m retentionInfoMessage) JSON() string {
	if m.Err != nil {
		m.Status = "failure"
	}
	msgBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Show retention info for a single object or version
func infoRetentionSingle(ctx context.Context, alias, url, versionID string) *probe.Error {
	newClnt, err := newClientFromAlias(alias, url)
	if err != nil {
		return err
	}

	msg := retentionInfoMessage{
		URLPath:   urlJoinPath(alias, url),
		VersionID: versionID,
	}

	mode, until, err := newClnt.GetObjectRetention(ctx, versionID)
	if err != nil {
		errResp := minio.ToErrorResponse(err.ToGoError())
		if errResp.Code != "NoSuchObjectLockConfiguration" {
			msg.Err = err.ToGoError()
			msg.Status = "failure"
			printMsg(msg)
			return err
		}
		err = nil
	}

	msg.Status = "success"
	msg.Mode = mode
	msg.Until = until

	printMsg(msg)
	return err
}

// Get Retention for one object/version or many objects within a given prefix.
func getRetention(ctx context.Context, target, versionID string, timeRef time.Time, withOlderVersions, isRecursive bool) error {
	clnt, err := newClient(target)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	// Quit early if urlStr does not point to an S3 server
	switch clnt.(type) {
	case *S3Client:
	default:
		fatal(errDummy().Trace(), "Retention is supported only for S3 servers.")
	}

	alias, urlStr, _ := mustExpandAlias(target)
	if versionID != "" || !isRecursive && !withOlderVersions {
		err := infoRetentionSingle(ctx, alias, urlStr, versionID)
		if err != nil {
			return exitStatus(globalErrorExitStatus)
		}
		return nil
	}

	lstOptions := ListOptions{isRecursive: isRecursive, showDir: DirNone}
	if !timeRef.IsZero() {
		lstOptions.withOlderVersions = withOlderVersions
		lstOptions.withDeleteMarkers = true
		lstOptions.timeRef = timeRef
	}

	var cErr error
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
		err := infoRetentionSingle(ctx, alias, content.URL.String(), content.VersionID)
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Invalid URL")
			cErr = exitStatus(globalErrorExitStatus)
			continue
		}
	}
	return cErr
}

// main for retention info command.
func mainRetentionInfo(cliCtx *cli.Context) error {
	ctx, cancelSetRetention := context.WithCancel(globalContext)
	defer cancelSetRetention()

	console.SetColor("RetentionSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("RetentionExpired", color.New(color.FgRed, color.Bold))
	console.SetColor("RetentionFailure", color.New(color.FgYellow))

	if len(cliCtx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(cliCtx, "info", 1)
	}

	target, versionID, recursive, rewind, withVersions := parseInfoRetentionArgs(cliCtx)

	if withVersions && rewind.IsZero() {
		rewind = time.Now().UTC()
	}

	return getRetention(ctx, target, versionID, rewind, withVersions, recursive)
}
