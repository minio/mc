/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
)

var retentionCmd = cli.Command{
	Name:   "retention",
	Usage:  "set object retention for objects with a given prefix",
	Action: mainRetention,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [governance | compliance] [VALIDITY]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
VALIDITY:
  This argument must be formatted like Nd or Ny where 'd' denotes days and 'y' denotes years e.g. 10d, 3y.

EXAMPLES:
   1. Set object retention for objects in a given prefix
     $ {{.HelpName}} myminio/mybucket/prefix compliance 30d
`,
}

// Structured message depending on the type of console.
type retentionCmdMessage struct {
	Mode     minio.RetentionMode `json:"mode"`
	Validity *string             `json:"validity"`
	URLPath  string              `json:"urlpath"`
	Status   string              `json:"status"`
	Err      error               `json:"error"`
}

// Colorized message for console printing.
func (m retentionCmdMessage) String() string {
	if m.Err != nil {
		return console.Colorize("RetentionMessageFailure", "Cannot set object retention on `"+m.URLPath+"`."+m.Err.Error())
	}
	return ""
}

// JSON'ified message for scripting.
func (m retentionCmdMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// setRetention - Set Retention for all objects within a given prefix.
func setRetention(urlStr string, mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit) error {
	clnt, err := newClient(urlStr)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	validityStr := func() *string {
		if validity == nil {
			return nil
		}

		unitStr := "d"
		if *unit == minio.Years {
			unitStr = "y"
		}
		s := fmt.Sprint(*validity, unitStr)
		return &s
	}

	var cErr error
	errorsFound := false
	for content := range clnt.List(true, false, false, DirNone) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}
		probeErr := clnt.PutObjectRetention(content.URL.Path, mode, validity, unit)
		if probeErr != nil {
			errorsFound = true
			printMsg(retentionCmdMessage{
				Mode:     *mode,
				Validity: validityStr(),
				Status:   "failure",
				URLPath:  content.URL.Path,
				Err:      probeErr.ToGoError(),
			})
		} else {
			if globalJSON {
				printMsg(retentionCmdMessage{
					Mode:     *mode,
					Validity: validityStr(),
					Status:   "success",
					URLPath:  content.URL.Path,
				})
			}
		}
	}
	if cErr == nil && !globalJSON {
		if errorsFound {
			console.Print(console.Colorize("RetentionPartialFailure", fmt.Sprintf("Errors found while setting retention on objects with prefix `%s`.\n", urlStr)))
		} else {
			console.Print(console.Colorize("RetentionSuccess", fmt.Sprintf("Object retention successfully set for prefix `%s`.\n", urlStr)))
		}
	}
	return cErr
}

// main for retention command.
func mainRetention(ctx *cli.Context) error {
	console.SetColor("RetentionSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("RetentionPartialFailure", color.New(color.FgRed, color.Bold))
	console.SetColor("RetentionMessageFailure", color.New(color.FgYellow))

	// Parse encryption keys per command.
	_, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// lock specific flags.
	clearLock := ctx.Bool("clear")

	args := ctx.Args()

	var urlStr string
	var mode *minio.RetentionMode
	var validity *uint
	var unit *minio.ValidityUnit

	switch l := len(args); l {
	case 3:
		urlStr = args[0]
		if clearLock {
			fatalIf(probe.NewError(errors.New("invalid argument")), "clear flag must be passed with target alone")
		}

		m := minio.RetentionMode(strings.ToUpper(args[1]))
		if !m.IsValid() {
			fatalIf(probe.NewError(errors.New("invalid argument")), "invalid retention mode '%v'", m)
		}

		mode = &m

		validityStr := args[2]
		unitStr := string(validityStr[len(validityStr)-1])

		validityStr = validityStr[:len(validityStr)-1]
		ui64, err := strconv.ParseUint(validityStr, 10, 64)
		if err != nil {
			fatalIf(probe.NewError(errors.New("invalid argument")), "invalid validity '%v'", args[2])
		}
		u := uint(ui64)
		validity = &u

		switch unitStr {
		case "d", "D":
			d := minio.Days
			unit = &d
		case "y", "Y":
			y := minio.Years
			unit = &y
		default:
			fatalIf(probe.NewError(errors.New("invalid argument")), "invalid validity format '%v'", args[2])
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "retention", 1)
	}
	return setRetention(urlStr, mode, validity, unit)
}
