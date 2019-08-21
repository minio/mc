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
	"net/url"
	"sort"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var (
	adminSpeedTestNetFlags = []cli.Flag{}
)

var adminSpeedTestNetCmd = cli.Command{
	Name:            "net",
	Usage:           "network performance of all cluster nodes",
	Action:          mainAdminSpeedTestNet,
	Before:          setGlobalsFromContext,
	Flags:           append(adminSpeedTestNetFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `Name:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get network performance of all cluster nodes by 'myminio' server.
     $ {{.HelpName}} myminio/
`,
}

// speedTestNetMessage holds network performance information.
type speedTestNetMessage struct {
	Addr         string               `json:"address"`
	Err          string               `json:"error"`
	SpeedTestNet []madmin.NetPerfInfo `json:"speedTestNet"`
}

func (s speedTestNetMessage) JSON() string {
	if len(s.SpeedTestNet) == 0 {
		s.Err = "cannot fetch network performance stats"
	}

	statusJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")

	return string(statusJSONBytes)
}

func (s speedTestNetMessage) String() string {
	if len(s.SpeedTestNet) == 0 {
		msg := fmt.Sprintf("%s: ", console.Colorize(monitorDegraded, s.Addr)) +
			fmt.Sprintf("Error: %s", console.Colorize(monitorFail, "cannot fetch network performance stats"))
		return msg
	}

	var msg string
	for _, destPerf := range s.SpeedTestNet {
		msg += fmt.Sprintf("%s -> %s: ", console.Colorize(monitor, s.Addr), destPerf.Addr)
		if destPerf.Error != "" {
			msg += fmt.Sprintf("Error: %v\n", destPerf.Error)
		} else {
			msg += fmt.Sprintf("%v/S\n", humanize.IBytes(destPerf.ReadThroughput))
		}
	}

	return msg
}

// checkAdminSpeedTestNetSyntax - validate all the passed arguments
func checkAdminSpeedTestNetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {

		exit := globalErrorExitStatus
		cli.ShowCommandHelpAndExit(ctx, "net", exit)
	}
}

func mainAdminSpeedTestNet(ctx *cli.Context) error {
	checkAdminSpeedTestNetSyntax(ctx)

	// set the console colors
	console.SetColor(monitor, color.New(color.FgGreen, color.Bold))
	console.SetColor(monitorDegraded, color.New(color.FgYellow, color.Bold))
	console.SetColor(monitorFail, color.New(color.FgRed, color.Bold))

	// Get the alias
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO admin client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	processErr := func(e error) error {
		switch e.(type) {
		case *json.SyntaxError, *url.Error:
			errMsg := ""
			if err != nil {
				errMsg = e.Error()
			}

			printMsg(speedTestNetMessage{
				Addr: aliasedURL,
				Err:  errMsg,
			})
		default:
			// If the error is not nil and unrecognized, just print it and exit
			fatalIf(probe.NewError(e), "Cannot get service status.")
		}

		return e
	}

	// Fetch network performance of all cluster nodes (all MinIO server instances)
	info, e := client.NetPerfInfo(madmin.DefaultNetPerfSize)
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	servers := []string{}
	for server := range info {
		servers = append(servers, server)
	}
	sort.Strings(servers)

	for _, server := range servers {
		printMsg(speedTestNetMessage{
			Addr:         server,
			SpeedTestNet: info[server],
		})
	}

	return nil
}
