/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	lhInfoFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply legal hold recursively",
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
var legalHoldInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "show legal hold info for object(s)",
	Action: mainLegalHoldInfo,
	Before: setGlobalsFromContext,
	Flags:  append(lhInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
   1. Show legal hold on a specific object
      $ {{.HelpName}} myminio/mybucket/prefix/obj.csv

   2. Show legal hold on a specific object version
      $ {{.HelpName}} myminio/mybucket/prefix/obj.csv --version-id "HiMFUTOowG6ylfNi4LKxD3ieHbgfgrvC"

   3. Show object legal hold recursively for all objects at a prefix
      $ {{.HelpName}} myminio/mybucket/prefix --recursive

   4. Show object legal hold recursively for all objects versions older than one year
      $ {{.HelpName}} myminio/mybucket/prefix --recursive --rewind 365d --versions
 `,
}

// Structured message depending on the type of console.
type legalHoldInfoMessage struct {
	LegalHold minio.LegalHoldStatus `json:"legalhold"`
	URLPath   string                `json:"urlpath"`
	VersionID string                `json:"versionID"`
	Status    string                `json:"status"`
	Err       error                 `json:"error,omitempty"`
}

// Colorized message for console printing.
func (l legalHoldInfoMessage) String() string {
	if l.Err != nil {
		return console.Colorize("LegalHoldMessageFailure", "Cannot get object legal hold status `"+l.URLPath+"`. "+l.Err.Error())
	}
	var msg string
	msg += "Object: " + l.URLPath
	if l.VersionID != "" {
		msg += ", Version id: " + l.VersionID
	}
	msg += ", "
	if l.LegalHold == "" {
		msg += "No legalhold set"
	} else {
		msg += "Legalhold: " + string(l.LegalHold)
	}
	return console.Colorize("LegalHoldSuccess", msg)
}

// JSON'ified message for scripting.
func (l legalHoldInfoMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// showLegalHoldInfo - show legalhold for one or many objects within a given prefix, with or without versioning
func showLegalHoldInfo(urlStr, versionID string, timeRef time.Time, withOlderVersions, recursive bool) error {
	ctx, cancelLegalHold := context.WithCancel(globalContext)
	defer cancelLegalHold()

	clnt, err := newClient(urlStr)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}
	if !recursive {
		lhold, err := clnt.GetObjectLegalHold(ctx, versionID)
		if err != nil {
			fatalIf(err.Trace(urlStr), "Failed to show legal hold information of `"+urlStr+"`.")
		} else {
			printMsg(legalHoldInfoMessage{
				LegalHold: lhold,
				Status:    "success",
				URLPath:   urlStr,
				VersionID: versionID,
			})
		}
		return nil
	}

	alias, _, _ := mustExpandAlias(urlStr)
	var cErr error
	errorsFound := false
	objectsFound := false
	lstOptions := ListOptions{isRecursive: true, showDir: DirNone}
	if !timeRef.IsZero() {
		lstOptions.withOlderVersions = withOlderVersions
		lstOptions.withDeleteMarkers = true
		lstOptions.timeRef = timeRef
	}
	for content := range clnt.List(ctx, lstOptions) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}
		objectsFound = true
		newClnt, perr := newClientFromAlias(alias, content.URL.String())
		if perr != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Invalid URL")
			continue
		}
		lhold, probeErr := newClnt.GetObjectLegalHold(ctx, content.VersionID)
		if probeErr != nil {
			errorsFound = true
			errorIf(probeErr.Trace(content.URL.Path), "Failed to get legal hold information on `"+content.URL.Path+"`")
		} else {
			if !globalJSON {
				printMsg(legalHoldInfoMessage{
					LegalHold: lhold,
					Status:    "success",
					URLPath:   content.URL.Path,
					VersionID: content.VersionID,
				})
			}
		}
	}

	if cErr == nil && !globalJSON {
		switch {
		case errorsFound:
			console.Print(console.Colorize("LegalHoldPartialFailure", fmt.Sprintf("Errors found while getting legal hold status on objects with prefix `%s`. \n", urlStr)))
		case !objectsFound:
			console.Print(console.Colorize("LegalHoldMessageFailure", fmt.Sprintf("No objects/versions found while getting legal hold status with prefix `%s`. \n", urlStr)))
		}
	}
	return cErr
}

// main for legalhold info command.
func mainLegalHoldInfo(ctx *cli.Context) error {
	console.SetColor("LegalHoldSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("LegalHoldPartialFailure", color.New(color.FgRed, color.Bold))
	console.SetColor("LegalHoldMessageFailure", color.New(color.FgYellow))

	targetURL, versionID, timeRef, recursive, withVersions := parseLegalHoldArgs(ctx)
	if timeRef.IsZero() && withVersions {
		timeRef = time.Now().UTC()
	}

	return showLegalHoldInfo(targetURL, versionID, timeRef, withVersions, recursive)
}
