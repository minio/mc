/*
 * Minio Client (C) 2019 Minio, Inc.
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
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
	xnet "github.com/minio/minio/pkg/net"
)

var (
	adminMonitorFlags = []cli.Flag{}
)

const (
	monitor         = "BoldGreen"
	monitorDegraded = "BoldYellow"
	monitorFail     = "BoldRed"
)

var adminMonitorCmd = cli.Command{
	Name:            "monitor",
	Usage:           "monitor cpu and mem statistics",
	Action:          mainAdminMonitor,
	Before:          setGlobalsFromContext,
	Flags:           append(adminMonitorFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `Name:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server cpu and mem statistics of the 'play' server.
       $ {{.HelpName}} play/
`,
}

// serverMonitorMessage holds service status info along with
// cpu, mem, net and disk monitoristics
type serverMonitorMessage struct {
	Monitor  string                     `json:"status"`
	Service  string                     `json:"service"`
	Addr     string                     `json:"address"`
	Err      string                     `json:"error"`
	CPULoad  *madmin.ServerCPULoadInfo  `json:"cpu,omitempty"`
	MemUsage *madmin.ServerMemUsageInfo `json:"mem,omitempty"`
	NetStats *madmin.ServerNetStatsInfo `json:"net, omitempty"`
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

	longest := 0

	leftField := 0
	midField := 0

	for i := range s.CPULoad.Load {
		load := s.CPULoad.Load[i]
		currLen := len(fmt.Sprintf("%.2f%% %.2f%% %.2f%%", load.Min, load.Avg, load.Max))
		if currLen > longest {
			longest = currLen
			leftField = len(fmt.Sprintf("%.2f%%", load.Min))
			midField = len(fmt.Sprintf("%.2f%%", load.Avg))
		}

		load = s.CPULoad.HistoricLoad[i]
		currLen = len(fmt.Sprintf("%.2f%% %.2f%% %.2f%%", load.Min, load.Avg, load.Max))
		if currLen > longest {
			longest = currLen
			leftField = len(fmt.Sprintf("%.2f%%", load.Min))
			midField = len(fmt.Sprintf("%.2f%%", load.Avg))
		}
	}

	min := "min"
	avg := "avg"
	max := "max"
	if len(min) < leftField {
		count := leftField - len(min)
		for i := 0; i < count; i++ {
			min = min + " "
		}
	}
	if len(avg) < midField {
		count := midField - len(avg)
		for i := 0; i < count; i++ {
			avg = avg + " "
		}
	}

	msg += fmt.Sprintf("%s        %s    %s    %s\n", console.Colorize(monitor, "   CPU"), min, avg, max)
	for i := range s.CPULoad.Load {
		load := s.CPULoad.Load[i]
		min := fmt.Sprintf("%.2f%%", load.Min)
		avg := fmt.Sprintf("%.2f%%", load.Avg)
		max := fmt.Sprintf("%.2f%%", load.Max)
		if len(min) < leftField {
			count := leftField - len(min)
			for i := 0; i < count; i++ {
				min = min + " "
			}
		}
		if len(avg) < midField {
			count := midField - len(avg)
			for i := 0; i < count; i++ {
				avg = avg + " "
			}
		}
		msg += fmt.Sprintf("%s    %s    %s    %s\n", "   current", min, avg, max)
		if len(s.CPULoad.HistoricLoad) > i {
			load := s.CPULoad.HistoricLoad[i]
			min := fmt.Sprintf("%.2f%%", load.Min)
			avg := fmt.Sprintf("%.2f%%", load.Avg)
			max := fmt.Sprintf("%.2f%%", load.Max)
			if len(min) < leftField {
				count := leftField - len(min)
				for i := 0; i < count; i++ {
					min = min + " "
				}
			}
			if len(avg) < midField {
				count := midField - len(avg)
				for i := 0; i < count; i++ {
					avg = avg + " "
				}
			}
			msg += fmt.Sprintf("%s   %s    %s    %s\n", "   historic", min, avg, max)
		}
		msg += "\n"
	}

	// Mem section
	msg += fmt.Sprintf("%s        usage\n", console.Colorize(monitor, "   MEM"))
	for i := range s.MemUsage.Usage {
		msg += fmt.Sprintf("   current    %s\n", humanize.IBytes(s.MemUsage.Usage[i].Mem))
		if len(s.MemUsage.HistoricUsage) > i {
			msg += fmt.Sprintf("   historic   %s\n", humanize.IBytes(s.MemUsage.HistoricUsage[i].Mem))
		}
		msg += "\n"
	}

	rx := 0

	for i := range s.NetStats.CurrentStats {
		tput := s.NetStats.CurrentStats[i]
		rxLen := len(fmt.Sprintf("%s", humanize.IBytes(tput.TotalReceived)))
		if rxLen > rx {
			rx = rxLen
		}

		tput = s.NetStats.OneMinWindow[i]
		rxLen = len(fmt.Sprintf("%s", humanize.IBytes(tput.TotalReceived)))
		if rxLen > rx {
			rx = rxLen
		}

		tput = s.NetStats.FiveMinWindow[i]
		rxLen = len(fmt.Sprintf("%s", humanize.IBytes(tput.TotalReceived)))
		if rxLen > rx {
			rx = rxLen
		}

		tput = s.NetStats.FifteenMinWindow[i]
		rxLen = len(fmt.Sprintf("%s", humanize.IBytes(tput.TotalReceived)))
		if rxLen > rx {
			rx = rxLen
		}
	}

	rxString := "rx"
	txString := "tx"
	if len(rxString) < rx {
		count := rx - len(rxString)
		for i := 0; i < count; i++ {
			rxString = rxString + " "
		}
	}

	// Net section
	msg += fmt.Sprintf("%s          %s    %s (internode only) \n", console.Colorize(monitor, "   NET"), rxString, txString)
	for i := range s.NetStats.CurrentStats {
		curr := s.NetStats.CurrentStats[i]
		oneMin := s.NetStats.OneMinWindow[i]
		fiveMin := s.NetStats.FiveMinWindow[i]
		fifteenMin := s.NetStats.FifteenMinWindow[i]

		msg += buildNetStats(curr, "current", rx)
		msg += buildNetStats(oneMin, "1min   ", rx)
		msg += buildNetStats(fiveMin, "5min   ", rx)
		msg += buildNetStats(fifteenMin, "15min  ", rx)

		msg = msg + "\n"
	}

	return msg
}

func buildNetStats(stat xnet.Stats, t string, rx int) string {
	tput := stat

	rxString := humanize.IBytes(tput.TotalReceived)
	txString := humanize.IBytes(tput.TotalTransmitted)
	if len(rxString) < rx {
		count := rx - len(rxString)
		for i := 0; i < count; i++ {
			rxString = rxString + " "
		}
	}
	msg := ""
	msg += fmt.Sprintf("   %s      %s    %s\n", t, rxString, txString)

	return msg
}

func mainAdminMonitor(ctx *cli.Context) error {
	checkAdminMonitorSyntax(ctx)

	// set the console colors
	console.SetColor(monitor, color.New(color.FgBlue, color.Bold))
	console.SetColor(monitorDegraded, color.New(color.FgYellow, color.Bold))
	console.SetColor(monitorFail, color.New(color.FgRed, color.Bold))

	// Get the alias
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new minio admin client
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
	// Fetch info of all CPU loads (all minio server instances)
	cpuLoads, e := client.ServerCPULoadInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	sort.Stable(&sortWrapper{cpuLoads: cpuLoads})

	memUsages, e := client.ServerMemUsageInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}
	sort.Stable(&sortWrapper{memUsages: memUsages})

	netStatss, e := client.ServerNetStatsInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}
	sort.Stable(&sortWrapper{netStatss: netStatss})

	for i, cpuLoad := range cpuLoads {
		memUsage := memUsages[i]
		netStats := netStatss[i]
		if cpuLoad.Error != "" {
			printMsg(serverMonitorMessage{
				Service: "off",
				Addr:    cpuLoad.Addr,
				Err:     cpuLoad.Error,
			})
			continue
		}
		if memUsage.Error != "" {
			printMsg(serverMonitorMessage{
				Service: "off",
				Addr:    memUsage.Addr,
				Err:     memUsage.Error,
			})
			continue
		}
		if netStats.Error != "" {
			printMsg(serverMonitorMessage{
				Service: "off",
				Addr:    memUsage.Addr,
				Err:     memUsage.Error,
			})
			continue
		}

		printMsg(serverMonitorMessage{
			Service:  "on",
			Addr:     cpuLoad.Addr,
			CPULoad:  &cpuLoad,
			MemUsage: &memUsage,
			NetStats: &netStats,
		})
	}
	return nil
}

type sortWrapper struct {
	cpuLoads  []madmin.ServerCPULoadInfo
	memUsages []madmin.ServerMemUsageInfo
	netStatss []madmin.ServerNetStatsInfo
}

func (s *sortWrapper) Len() int {
	if s.cpuLoads != nil {
		return len(s.cpuLoads)
	}
	if s.memUsages != nil {
		return len(s.memUsages)
	}
	return len(s.netStatss)
}

func (s *sortWrapper) Swap(i, j int) {
	if s.cpuLoads != nil {
		s.cpuLoads[i], s.cpuLoads[j] = s.cpuLoads[j], s.cpuLoads[i]
		return
	}
	if s.memUsages != nil {
		s.memUsages[i], s.memUsages[j] = s.memUsages[j], s.memUsages[i]
	}
	if s.netStatss != nil {
		s.netStatss[i], s.netStatss[j] = s.netStatss[j], s.netStatss[i]
	}
}

func (s *sortWrapper) Less(i, j int) bool {
	if s.cpuLoads != nil {
		return strings.Compare(s.cpuLoads[i].Addr, s.cpuLoads[j].Addr) < 0
	}
	if s.memUsages != nil {
		return strings.Compare(s.memUsages[i].Addr, s.memUsages[j].Addr) < 0
	}
	if s.netStatss != nil {
		return strings.Compare(s.netStatss[i].Addr, s.netStatss[j].Addr) < 0
	}
	return false
}

// checkAdminMonitorSyntax - validate all the passed arguments
func checkAdminMonitorSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {

		exit := globalErrorExitStatus
		cli.ShowCommandHelpAndExit(ctx, "monitor", exit)
	}
}
