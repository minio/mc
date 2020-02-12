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
	"encoding/json"
	"strings"
	"errors"
	
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminOBDFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "tests",
		Usage: "diagnostics type, possible values are 'all', 'drive', 'net', 'sysinfo', 'hwinfo' and 'config'",
		Value: "all",
	},
}


var adminOBDCmd = cli.Command{
	Name:   "obd",
	Usage:  "run on-board diagnostics",
	Action: mainAdminOBD,
	Before: setGlobalsFromContext,
	Flags:  append(adminOBDFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server information of the 'play' MinIO server.
     {{.Prompt}} {{.HelpName}} play/
`,
}

type clusterOBDStruct struct {
	Status string         `json:"status"`
	Error  string         `json:"error,omitempty"`
	Info   madmin.OBDInfo `json:"info,omitempty"`
}

func (u clusterOBDStruct) String() string {
	data, err := json.Marshal(u)
	if err != nil {
		fatalIf(probe.NewError(err), "unable to marshal into JSON.")
	}
	return string(data)
}

// JSON jsonifies service status message.
func (u clusterOBDStruct) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(u, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate arguments passed by a user
func checkAdminOBDSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "obd", 1) // last argument is exit code
	}
}

func mainAdminOBD(ctx *cli.Context) error {
	checkAdminOBDSyntax(ctx)

	drive, net, sysinfo, hwinfo, config := false, false, false, false, false
	
	types := ctx.String("tests")
	obds := strings.Split(types, ",")
	for _, obd := range obds {
		switch obd {
		case "all":
			drive, net, sysinfo, hwinfo, config = true, true, true, true, true
		case "drive":
			drive = true
		case "net":
			net = true
		case "sysinfo":
			sysinfo = true
		case "hwinfo":
			hwinfo = true
		case "config":
			config = true
		default:
			fatalIf(probe.NewError(errors.New("unrecognized --tests option")), obd)
		}
	}
	
	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	var clusterOBDInfo clusterOBDStruct
	// Fetch info of all servers (cluster or single server)
	adminOBDInfo, e := client.ServerOBDInfo(drive, net, sysinfo, hwinfo, config)
	if e != nil {
		clusterOBDInfo.Status = "error"
		clusterOBDInfo.Error = e.Error()
	} else {
		clusterOBDInfo.Status = "success"
		clusterOBDInfo.Error = ""
	}
	clusterOBDInfo.Info = adminOBDInfo

	printMsg(clusterOBDStruct(clusterOBDInfo))

	return nil
}
