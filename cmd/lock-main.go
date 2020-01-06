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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio/pkg/console"
)

var (
	lockFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "clear, c",
			Usage: "clears previously stored object lock configuration",
		},
	}
)

var lockCmd = cli.Command{
	Name:   "lock",
	Usage:  "set and get object lock configuration",
	Action: mainLock,
	Before: setGlobalsFromContext,
	Flags:  append(lockFlags, globalFlags...),
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
   1. Set object lock configuration
     $ {{.HelpName}} myminio/mybucket compliance 30d

   2. Get object lock configuration
     $ {{.HelpName}} myminio/mybucket

   3. Clear object lock configuration
     $ {{.HelpName}} --clear myminio/mybucket
`,
}

// Structured message depending on the type of console.
type lockCmdMessage struct {
	Enabled  string               `json:"enabled"`
	Mode     *minio.RetentionMode `json:"mode"`
	Validity *string              `json:"validity"`
	Status   string               `json:"status"`
}

// Colorized message for console printing.
func (m lockCmdMessage) String() string {
	if m.Mode == nil {
		return fmt.Sprintf("No object lock configuration is enabled")
	}

	return fmt.Sprintf("%s mode is enabled for %s", console.Colorize("Mode", *m.Mode), console.Colorize("Validity", *m.Validity))
}

// JSON'ified message for scripting.
func (m lockCmdMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// lock - set/get object lock configuration.
func lock(urlStr string, mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit, clearLock bool) error {
	client, err := newClient(urlStr)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	s3Client, ok := client.(*s3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
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

	if clearLock || mode != nil {
		err = s3Client.SetObjectLockConfig(mode, validity, unit)
		fatalIf(err, "Cannot enable object lock configuration on the specified bucket.")
	} else {
		mode, validity, unit, err = s3Client.GetObjectLockConfig()
		fatalIf(err, "Cannot get object lock configuration on the specified bucket.")
	}

	printMsg(lockCmdMessage{
		Enabled:  "Enabled",
		Mode:     mode,
		Validity: validityStr(),
		Status:   "success",
	})

	return nil
}

// main for lock command.
func mainLock(ctx *cli.Context) error {
	console.SetColor("Mode", color.New(color.FgCyan, color.Bold))
	console.SetColor("Validity", color.New(color.FgYellow))

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
	case 1:
		urlStr = args[0]

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
		cli.ShowCommandHelpAndExit(ctx, "lock", 1)
	}

	return lock(urlStr, mode, validity, unit, clearLock)
}
