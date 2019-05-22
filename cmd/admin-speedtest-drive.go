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
	"fmt"
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var (
	adminSpeedTestDriveFlags = []cli.Flag{}
)

var adminSpeedTestDriveCmd = cli.Command{
	Name:   "drive",
	Usage:  "Test the read and write speed of the disks on a MinIO setup",
	Action: mainAdminSpeedTestDrive,
	Before: setGlobalsFromContext,
	Flags:  append(adminSpeedTestDriveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Test read and write speeds of the disks on MinIO play setup
       $ {{.HelpName}} play/
`,
}

// speedTestDrivesMsg is a wrapper type to implement the message
// interface for printing command output in plain text and JSON
// formats
type speedTestDrivesMsg madmin.SpeedTestResultItem

func (stdm speedTestDrivesMsg) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Node: %s\n", stdm.Node)
	fmt.Fprintf(&b, "Drive: %s\n", stdm.Drive)
	if stdm.Err != "" {
		fmt.Fprint(&b, "Err: %s\n", stdm.Err)
	} else {
		fmt.Fprintf(&b, "WriteSpeed (MiB/s): %f\n", stdm.WriteSpeed)
		fmt.Fprintf(&b, "ReadSpeed: (MiB/s): %f\n", stdm.ReadSpeed)
	}
	return b.String()
}

func (stdm speedTestDrivesMsg) JSON() string {
	msgJSONBytes, e := json.MarshalIndent(stdm, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")

	return string(msgJSONBytes)
}

// checkAdminSpeedTestDriveSyntax - validate all passed arguments
func checkAdminSpeedTestDriveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "drive", 1) // last argument is exit code
	}
}

func mainAdminSpeedTestDrive(ctx *cli.Context) error {
	checkAdminSpeedTestDriveSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	doneCh := make(chan bool)
	defer close(doneCh)
	resultCh, stdErr := client.SpeedTestDrives(doneCh)
	fatalIf(probe.NewError(stdErr), "Could not get speed test result on drives.")

	for res := range resultCh {
		printMsg(speedTestDrivesMsg(res))
	}

	return nil
}
