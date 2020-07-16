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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
)

var (
	lhFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply legal hold recursively",
		},
	}
)
var legalHoldCmd = cli.Command{
	Name:   "legalhold",
	Usage:  "set legal hold for object(s)",
	Action: mainLegalHold,
	Before: setGlobalsFromContext,
	Flags:  append(lhFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
 
USAGE:
  {{.HelpName}} [FLAGS] TARGET [ON | OFF]
 
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
 
EXAMPLES:
   1. Enable legal hold on a specific object
      $ {{.HelpName}} myminio/mybucket/prefix/obj.csv ON

   2. Enable object legal hold recursively for all objects at a prefix
      $ {{.HelpName}} myminio/mybucket/prefix ON --recursive
 `,
}

// Structured message depending on the type of console.
type legalHoldCmdMessage struct {
	LegalHold minio.LegalHoldStatus `json:"legalhold"`
	URLPath   string                `json:"urlpath"`
	Status    string                `json:"status"`
	Err       error                 `json:"error,omitempty"`
}

// Colorized message for console printing.
func (l legalHoldCmdMessage) String() string {
	if l.Err != nil {
		return console.Colorize("LegalHoldMessageFailure", "Cannot set object legal hold status `"+l.URLPath+"`. "+l.Err.Error())
	}
	return console.Colorize("LegalHoldSuccess", fmt.Sprintf("Object legal hold successfully set for `%s`.", l.URLPath))
}

// JSON'ified message for scripting.
func (l legalHoldCmdMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// setRetention - Set Retention for all objects within a given prefix.
func setLegalHold(urlStr string, lhold minio.LegalHoldStatus, isRecursive bool) error {
	ctx, cancelLegalHold := context.WithCancel(globalContext)
	defer cancelLegalHold()

	clnt, err := newClient(urlStr)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}
	if !isRecursive {
		err = clnt.PutObjectLegalHold(ctx, lhold)
		if err != nil {
			errorIf(err.Trace(urlStr), "Failed to set legal hold on `"+urlStr+"` successfully")
		} else {
			printMsg(legalHoldCmdMessage{
				LegalHold: lhold,
				Status:    "success",
				URLPath:   urlStr,
			})
		}
		return nil
	}

	alias, _, _ := mustExpandAlias(urlStr)
	var cErr error
	errorsFound := false
	for content := range clnt.List(ctx, isRecursive, false, false, DirNone) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}

		newClnt, perr := newClientFromAlias(alias, content.URL.String())
		if perr != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Invalid URL")
			continue
		}
		probeErr := newClnt.PutObjectLegalHold(ctx, lhold)
		if probeErr != nil {
			errorsFound = true
			errorIf(probeErr.Trace(content.URL.Path), "Failed to set legal hold on `"+content.URL.Path+"` successfully")
		} else {
			if globalJSON {
				printMsg(legalHoldCmdMessage{
					LegalHold: lhold,
					Status:    "success",
					URLPath:   content.URL.Path,
				})
			}
		}
	}
	if cErr == nil && !globalJSON {
		if errorsFound {
			console.Print(console.Colorize("LegalHoldPartialFailure", fmt.Sprintf("Errors found while setting legal hold status on objects with prefix `%s`. \n", urlStr)))
		} else {
			console.Print(console.Colorize("LegalHoldSuccess", fmt.Sprintf("Object legal hold successfully set for prefix `%s`.\n", urlStr)))
		}
	}
	return cErr
}

// main for retention command.
func mainLegalHold(ctx *cli.Context) error {
	console.SetColor("LegalHoldSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("LegalHoldPartialFailure", color.New(color.FgRed, color.Bold))
	console.SetColor("LegalHoldMessageFailure", color.New(color.FgYellow))
	args := ctx.Args()

	var urlStr string
	var lhold minio.LegalHoldStatus
	switch l := len(args); l {
	case 2:
		urlStr = args[0]
		lhold = minio.LegalHoldStatus(strings.ToUpper(args[1]))
		if !lhold.IsValid() {
			fatalIf(errInvalidArgument().Trace(urlStr), "invalid legal hold status '%v'", lhold)
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "legalhold", 1)
	}
	return setLegalHold(urlStr, lhold, ctx.Bool("recursive"))
}
