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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
)

var (
	rFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply retention recursively",
		},
		cli.BoolFlag{
			Name:  bypass,
			Usage: "bypass governance",
		},
	}
)

var bypass = "bypass"

var retentionCmd = cli.Command{
	Name:   "retention",
	Usage:  "set retention for object(s)",
	Action: mainRetention,
	Before: initBeforeRunningCmd,
	Flags:  append(rFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [governance | compliance] VALIDITY

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
VALIDITY:
  This argument must be formatted like Nd or Ny where 'd' denotes days and 'y' denotes years e.g. 10d, 3y.

EXAMPLES:
  1. Set object retention for a specific object
     $ {{.HelpName}} myminio/mybucket/prefix/obj.csv compliance 30d

  2. Set object retention for recursively for all objects at a given prefix
     $ {{.HelpName}} myminio/mybucket/prefix compliance 30d  --recursive
`,
}

// Structured message depending on the type of console.
type retentionCmdMessage struct {
	Mode     minio.RetentionMode `json:"mode"`
	Validity string              `json:"validity"`
	URLPath  string              `json:"urlpath"`
	Status   string              `json:"status"`
	Err      error               `json:"error"`
}

// Colorized message for console printing.
func (m retentionCmdMessage) String() string {
	if m.Err != nil {
		return console.Colorize("RetentionMessageFailure", fmt.Sprintf("Cannot set object retention on `%s`: %s", m.URLPath, m.Err))
	}
	return console.Colorize("RetentionSuccess", fmt.Sprintf("Object retention successfully set for `%s`", m.URLPath))
}

// JSON'ified message for scripting.
func (m retentionCmdMessage) JSON() string {
	if m.Err != nil {
		m.Status = "failure"
	}
	msgBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func getRetainUntilDate(validity uint64, unit minio.ValidityUnit) (string, *probe.Error) {
	if validity == 0 {
		return "", probe.NewError(fmt.Errorf("invalid validity '%v'", validity))
	}
	t := UTCNow()
	if unit == minio.Years {
		t = t.AddDate(int(validity), 0, 0)
	} else {
		t = t.AddDate(0, 0, int(validity))
	}
	timeStr := t.Format(time.RFC3339)

	return timeStr, nil
}

// setRetention - Set Retention for all objects within a given prefix.
func setRetention(urlStr string, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, bypassGovernance, isRecursive bool) error {
	ctx, cancelSetRetention := context.WithCancel(globalContext)
	defer cancelSetRetention()

	clnt, err := newClient(urlStr)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	// Quit early if urlStr does not point to an S3 server
	switch clnt.(type) {
	case *fsClient:
		fatal(errDummy().Trace(), "Retention for filesystem not supported.")
	}

	alias, _, _ := mustExpandAlias(urlStr)

	var cErr error
	for content := range clnt.List(ctx, isRecursive, false, false, DirNone) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}
		timeStr, err := getRetainUntilDate(validity, unit)
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Invalid retention date")
			continue
		}
		retainUntil, e := time.Parse(time.RFC3339, timeStr)
		if e != nil {
			errorIf(probe.NewError(e).Trace(clnt.GetURL().String()), "Invalid retention date")
			continue
		}

		newClnt, err := newClientFromAlias(alias, content.URL.String())
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Invalid URL")
			continue
		}
		err = newClnt.PutObjectRetention(ctx, mode, retainUntil, bypassGovernance)
		if err != nil {
			printMsg(retentionCmdMessage{
				Mode:     mode,
				Validity: fmt.Sprintf("%d%s", validity, unit),
				Status:   "failure",
				URLPath:  urlJoinPath(alias, content.URL.Path),
				Err:      err.ToGoError(),
			})
		} else {
			printMsg(retentionCmdMessage{
				Mode:     mode,
				Validity: fmt.Sprintf("%d%s", validity, unit),
				Status:   "success",
				URLPath:  urlJoinPath(alias, content.URL.Path),
			})
		}
	}
	return cErr
}

// main for retention command.
func mainRetention(ctx *cli.Context) error {
	console.SetColor("RetentionSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("RetentionPartialFailure", color.New(color.FgRed, color.Bold))
	console.SetColor("RetentionMessageFailure", color.New(color.FgYellow))
	args := ctx.Args()

	var urlStr string
	var mode minio.RetentionMode
	var validity uint64
	var unit minio.ValidityUnit

	switch l := len(args); l {
	case 3:
		urlStr = args[0]
		mode = minio.RetentionMode(strings.ToUpper(args[1]))
		if !mode.IsValid() {
			fatalIf(errInvalidArgument().Trace(args...), "invalid retention mode '%v'", mode)
		}

		validityStr := args[2]
		unitStr := validityStr[len(validityStr)-1]
		validityStr = validityStr[:len(validityStr)-1]

		var e error
		validity, e = strconv.ParseUint(validityStr, 10, 64)
		if e != nil {
			fatalIf(probe.NewError(e).Trace(urlStr), "invalid validity '%v'", validityStr)
		}

		switch unitStr {
		case 'd', 'D':
			unit = minio.Days
		case 'y', 'Y':
			unit = minio.Years
		default:
			fatalIf(errInvalidArgument().Trace(urlStr), "invalid validity format '%v'", unitStr)
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "retention", 1)
	}
	return setRetention(urlStr, mode, validity, unit, ctx.Bool("bypass"), ctx.Bool("recursive"))
}
