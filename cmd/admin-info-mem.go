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

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

const (
	memory         = "BoldGreen"
	memoryDegraded = "BoldYellow"
	memoryFail     = "BoldRed"
)

var adminInfoMem = cli.Command{
	Name:   "mem",
	Usage:  "display MinIO server memory information",
	Action: mainAdminMemInfo,
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
     {{.Prompt}} {{.HelpName}} play/
`,
}

// serverMonitorMessage holds service status info along with
// cpu, mem, net and disk monitoristics
type serverMemoryInfo struct {
	Monitor  string                     `json:"status"`
	Service  string                     `json:"service"`
	Addr     string                     `json:"address"`
	Err      string                     `json:"error"`
	MemUsage *madmin.ServerMemUsageInfo `json:"mem,omitempty"`
}

func (s serverMemoryInfo) JSON() string {
	s.Monitor = "success"
	if s.Err != "" {
		s.Monitor = "fail"
	}
	statusJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")

	return string(statusJSONBytes)
}

func (s serverMemoryInfo) String() string {
	msg := ""
	dot := "●"

	if s.Err != "" || s.Service == "off" {
		msg += fmt.Sprintf("%s  %s\n", console.Colorize(memoryFail, dot), s.Addr)
		msg += fmt.Sprintf("   Server is %s\n", console.Colorize(memoryFail, "offline"))
		msg += fmt.Sprintf("   Error : %s\n", console.Colorize(memoryFail, s.Err))
		return msg
	}

	// Print server title
	msg += fmt.Sprintf("%s  %s    \n", console.Colorize(memory, dot), s.Addr)
	msg += "\n"

	// Mem section
	msg += fmt.Sprintf("%s        usage\n", console.Colorize(memory, "   MEM"))
	for i := range s.MemUsage.Usage {
		msg += fmt.Sprintf("   current    %s\n", humanize.IBytes(s.MemUsage.Usage[i].Mem))
		if len(s.MemUsage.HistoricUsage) > i {
			msg += fmt.Sprintf("   historic   %s\n", humanize.IBytes(s.MemUsage.HistoricUsage[i].Mem))
		}
		msg += "\n"
	}

	return msg
}

func mainAdminMemInfo(ctx *cli.Context) error {
	checkAdminMemInfoSyntax(ctx)

	// set the console colors
	console.SetColor(memory, color.New(color.FgGreen, color.Bold))
	console.SetColor(memoryDegraded, color.New(color.FgYellow, color.Bold))
	console.SetColor(memoryFail, color.New(color.FgRed, color.Bold))

	// Get the alias
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO admin client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

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

	memUsages, e := client.ServerMemUsageInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}
	sort.Sort(&sortMemWrapper{memUsages: memUsages})

	for _, memUsage := range memUsages {

		if memUsage.Error != "" {
			printMsg(serverMemoryInfo{
				Service: "off",
				Addr:    memUsage.Addr,
				Err:     memUsage.Error,
			})
			continue
		}

		printMsg(serverMemoryInfo{
			Service:  "on",
			Addr:     memUsage.Addr,
			MemUsage: &memUsage,
		})
	}
	return nil
}

type sortMemWrapper struct {
	memUsages []madmin.ServerMemUsageInfo
}

func (s *sortMemWrapper) Len() int {
	return len(s.memUsages)
}

func (s *sortMemWrapper) Swap(i, j int) {
	if s.memUsages != nil {
		s.memUsages[i], s.memUsages[j] = s.memUsages[j], s.memUsages[i]
	}
}

func (s *sortMemWrapper) Less(i, j int) bool {

	if s.memUsages != nil {
		return strings.Compare(s.memUsages[i].Addr, s.memUsages[j].Addr) < 0
	}
	return false

}

// checkAdminMonitorSyntax - validate all the passed arguments
func checkAdminMemInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {

		exit := globalErrorExitStatus
		cli.ShowCommandHelpAndExit(ctx, "mem", exit)
	}
}
