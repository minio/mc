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
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

const (
	monitor         = "BoldGreen"
	monitorDegraded = "BoldYellow"
	monitorFail     = "BoldRed"
)

var adminInfoCPU = cli.Command{
	Name:   "cpu",
	Usage:  "display MinIO server cpu information",
	Action: mainAdminCPUInfo,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server CPU information of the 'play' MinIO server.
     $ {{.HelpName}} play/
`,
}

// serverMonitorMessage holds service status info along with
// cpu, mem, net and disk monitoristics
type serverMonitorMessage struct {
	Monitor string                    `json:"status"`
	Service string                    `json:"service"`
	Addr    string                    `json:"address"`
	Err     string                    `json:"error"`
	CPULoad *madmin.ServerCPULoadInfo `json:"cpu,omitempty"`
}

func (s serverMonitorMessage) JSON() string {
	s.Monitor = "success"
	if s.Err != "" {
		s.Monitor = "fail"
	}
	statusJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")

	return string(statusJSONBytes)
}

func (s serverMonitorMessage) String() string {
	msg := ""
	dot := "●"

	if s.Err != "" || s.Service == "off" {
		msg += fmt.Sprintf("%s  %s\n", console.Colorize(monitorFail, dot), s.Addr)
		msg += fmt.Sprintf("   Server is %s\n", console.Colorize(monitorFail, "offline"))
		msg += fmt.Sprintf("   Error : %s\n", console.Colorize(monitorFail, s.Err))
		return msg
	}

	// Print server title
	msg += fmt.Sprintf("%s  %s    \n", console.Colorize(monitor, dot), s.Addr)
	msg += "\n"

	msg += fmt.Sprintf("%s        min        avg      max\n", console.Colorize(monitor, "   CPU"))
	for i := range s.CPULoad.Load {
		msg += fmt.Sprintf("   current    %.2f%%      %.2f%%    %.2f%%\n", s.CPULoad.Load[i].Min, s.CPULoad.Load[i].Avg, s.CPULoad.Load[i].Max)
		if len(s.CPULoad.HistoricLoad) > i {
			msg += fmt.Sprintf("   historic   %.2f%%      %.2f%%    %.2f%%\n", s.CPULoad.HistoricLoad[i].Min, s.CPULoad.HistoricLoad[i].Avg, s.CPULoad.HistoricLoad[i].Max)
		}
		msg += "\n"
	}

	return msg
}

func mainAdminCPUInfo(ctx *cli.Context) error {
	checkAdminCPUInfoSyntax(ctx)

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

	printOfflineErrorMessage := func(err error) {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		printMsg(serverMonitorMessage{
			Addr:    aliasedURL,
			Service: "off",
			Err:     errMsg,
		})
	}

	processErr := func(e error) error {
		switch e.(type) {
		case *json.SyntaxError:
			printOfflineErrorMessage(e)
			return e
		case *url.Error:
			printOfflineErrorMessage(e)
			return e
		default:
			// If the error is not nil and unrecognized, just print it and exit
			fatalIf(probe.NewError(e), "Cannot get service status.")
		}
		return nil
	}
	// Fetch info of all CPU loads (all MinIO server instances)
	cpuLoads, e := client.ServerCPULoadInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	sort.Stable(&sortCPUWrapper{cpuLoads: cpuLoads})

	for _, cpuLoad := range cpuLoads {
		if cpuLoad.Error != "" {
			printMsg(serverMonitorMessage{
				Service: "off",
				Addr:    cpuLoad.Addr,
				Err:     cpuLoad.Error,
			})
			continue
		}

		printMsg(serverMonitorMessage{
			Service: "on",
			Addr:    cpuLoad.Addr,
			CPULoad: &cpuLoad,
		})
	}
	return nil
}

type sortCPUWrapper struct {
	cpuLoads []madmin.ServerCPULoadInfo
}

func (s *sortCPUWrapper) Len() int {
	return len(s.cpuLoads)

}

func (s *sortCPUWrapper) Swap(i, j int) {
	if s.cpuLoads != nil {
		s.cpuLoads[i], s.cpuLoads[j] = s.cpuLoads[j], s.cpuLoads[i]
		return
	}
}

func (s *sortCPUWrapper) Less(i, j int) bool {
	if s.cpuLoads != nil {
		return strings.Compare(s.cpuLoads[i].Addr, s.cpuLoads[j].Addr) < 0
	}
	return false
}

// checkAdminMonitorSyntax - validate all the passed arguments
func checkAdminCPUInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {

		exit := globalErrorExitStatus
		cli.ShowCommandHelpAndExit(ctx, "cpu", exit)
	}
}
