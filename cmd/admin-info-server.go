/*
 * MinIO Client (C) 2016-2019 MinIO, Inc.
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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminInfoServer = cli.Command{
	Name:   "server",
	Usage:  "display MinIO server information",
	Action: mainAdminServerInfo,
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
  1. Get server information of the 'play' MinIO server.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// backendType - indicates the type of backend storage
type backendType string

const (
	fsType      = backendType("FS")
	erasureType = backendType("Erasure")
)

// fsBackend contains specific FS storage information
type fsBackend struct {
	Type backendType `json:"backendType"`
}

// xlBackend contains specific erasure storage information
type xlBackend struct {
	Type         backendType `json:"backendType"`
	OnlineDisks  int         `json:"onlineDisks"`
	OfflineDisks int         `json:"offlineDisks"`
	// Data disks for currently configured Standard storage class.
	StandardSCData int `json:"standardSCData"`
	// Parity disks for currently configured Standard storage class.
	StandardSCParity int `json:"standardSCParity"`
	// Data disks for currently configured Reduced Redundancy storage class.
	RRSCData int `json:"rrSCData"`
	// Parity disks for currently configured Reduced Redundancy storage class.
	RRSCParity int `json:"rrSCParity"`

	// List of all disk status.
	Sets [][]madmin.DriveInfo `json:"sets"`
}

// backendStatus represents the overall information of all backend storage types
type backendStatus struct {
	// Total used space per tenant.
	Used uint64 `json:"used"`
	// Total available space.
	Available uint64 `json:"available"`
	// Total disk space.
	Total uint64 `json:"total"`
	// Backend type.
	Backend interface{} `json:"backend"`
}

// ServerInfo holds the whole server information that will be
// returned by ServerInfo API.
type ServerInfo struct {
	ConnStats  madmin.ServerConnStats    `json:"network"`
	Properties madmin.ServerProperties   `json:"server"`
	CPULoad    madmin.ServerCPULoadInfo  `json:"cpu,omitempty"`
	MemUsage   madmin.ServerMemUsageInfo `json:"mem,omitempty"`
}

// infoMessage container to hold service status information.
type infoMessage struct {
	Status      string        `json:"status"`
	Service     string        `json:"service"`
	Addr        string        `json:"address"`
	Err         string        `json:"error"`
	StorageInfo backendStatus `json:"storage"`
	*ServerInfo
}

func filterPerNode(addr string, m map[string]int) int {
	if val, ok := m[addr]; ok {
		return val
	}
	return -1
}

// String colorized service status message.
func (u infoMessage) String() (msg string) {
	dot := "●"

	// When service is offline
	if u.Service == "off" {
		msg += fmt.Sprintf("%s  %s\n", console.Colorize("InfoFail", dot), console.Colorize("PrintB", u.Addr))
		msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("InfoFail", "offline"))
		return
	}

	// Print error if any and exit
	if u.Err != "" {
		msg += fmt.Sprintf("%s  %s\n", console.Colorize("InfoFail", dot), console.Colorize("PrintB", u.Addr))
		msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("InfoFail", "offline"))
		e := u.Err
		if strings.Trim(e, " ") == "rpc: retry error" {
			e = "unreachable"
		}
		msg += fmt.Sprintf("    Error: %s", console.Colorize("InfoFail", e))
		return
	}

	// Print server title
	msg += fmt.Sprintf("%s  %s\n", console.Colorize("Info", dot), console.Colorize("PrintB", u.Addr))

	// Uptime
	msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("Info",
		humanize.RelTime(time.Now(), time.Now().Add(-u.ServerInfo.Properties.Uptime), "", "")))

	// Version
	version := u.ServerInfo.Properties.Version
	if u.ServerInfo.Properties.Version == "DEVELOPMENT.GOGET" {
		version = "<development>"
	}
	msg += fmt.Sprintf("  Version: %s\n", version)
	// Region
	if u.ServerInfo.Properties.Region != "" {
		msg += fmt.Sprintf("   Region: %s\n", u.ServerInfo.Properties.Region)
	}
	// ARNs
	sqsARNs := ""
	for _, v := range u.ServerInfo.Properties.SQSARN {
		sqsARNs += fmt.Sprintf("%s ", v)
	}
	if sqsARNs != "" {
		msg += fmt.Sprintf(" SQS ARNs: %s\n", sqsARNs)
	}

	// Incoming/outgoing
	msg += fmt.Sprintf("  Storage: Used %s, Free %s",
		humanize.IBytes(u.StorageInfo.Used),
		humanize.IBytes(u.StorageInfo.Available))
	if v, ok := u.StorageInfo.Backend.(xlBackend); ok {
		upBackends := 0
		downBackends := 0
		for _, set := range v.Sets {
			for _, s := range set {
				if len(s.Endpoint) > 0 && (strings.Contains(s.Endpoint, u.Addr) || s.Endpoint[0] == '/' || s.Endpoint[0] == '.') {
					if s.State == "ok" {
						upBackends++
					} else {
						downBackends++
					}
				}
			}
		}
		upBackendsString := fmt.Sprintf("%d", upBackends)
		if downBackends != 0 {
			upBackendsString = console.Colorize("InfoFail", fmt.Sprintf("%d", upBackends))
		}
		msg += fmt.Sprintf("\n   Drives: %s/%d %s\n", upBackendsString,
			upBackends+downBackends, console.Colorize("Info", "OK"))
		msg += "\n"

		//CPU section
		msg += fmt.Sprintf("%s        min        avg      max\n", console.Colorize("Info", "   CPU"))
		for i := range u.CPULoad.Load {
			msg += fmt.Sprintf("   current    %.2f%%      %.2f%%    %.2f%%\n", u.CPULoad.Load[i].Min, u.CPULoad.Load[i].Avg, u.CPULoad.Load[i].Max)
			if len(u.CPULoad.HistoricLoad) > i {
				msg += fmt.Sprintf("   historic   %.2f%%      %.2f%%    %.2f%%\n", u.CPULoad.HistoricLoad[i].Min, u.CPULoad.HistoricLoad[i].Avg, u.CPULoad.HistoricLoad[i].Max)
			}
			msg += "\n"
		}

		// Mem section
		msg += fmt.Sprintf("%s        usage\n", console.Colorize("Info", "   MEM"))
		for i := range u.MemUsage.Usage {
			msg += fmt.Sprintf("   current    %s\n", humanize.IBytes(u.MemUsage.Usage[i].Mem))
			if len(u.MemUsage.HistoricUsage) > i {
				msg += fmt.Sprintf("   historic   %s\n", humanize.IBytes(u.MemUsage.HistoricUsage[i].Mem))
			}
			msg += "\n"
		}

	}
	return
}

// JSON jsonified service status message.
func (u infoMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate all the passed arguments
func checkAdminServerInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "server", 1) // last argument is exit code
	}
}

func mainAdminServerInfo(ctx *cli.Context) error {
	checkAdminServerInfoSyntax(ctx)

	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoDegraded", color.New(color.FgYellow, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	printOfflineErrorMessage := func(err error) {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		printMsg(infoMessage{
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

	// Fetch info of all servers (cluster or single server)
	serversInfo, e := client.ServerInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	// Fetch storage info of all servers (cluster or single server)
	storageInfo, e := client.StorageInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	// Fetch info of all CPU loads (all MinIO server instances)
	cpuLoads, e := client.ServerCPULoadInfo()

	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}
	memUsages, e := client.ServerMemUsageInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	infoMessages := []infoMessage{}

	// Checks if the corresponding mountPath is found for the the given nodeAddr.
	foundMatchingPath := func(mountPath, nodeAddr string) bool {
		// NOTE: XL/FS mountpaths will have `/` as the first character denoting the absolute disk path.
		return strings.Contains(mountPath, nodeAddr) || mountPath[0] == '/' || nodeAddr[0] == '/' || nodeAddr[0] == '.'
	}

	for i, serverInfo := range serversInfo {
		cpuLoad := cpuLoads[i]
		memUsage := memUsages[i]

		// Print the error if exists and jump to the next server
		if serverInfo.Error != "" {

			infoMessages = append(infoMessages, infoMessage{
				Service: "on",
				Addr:    serverInfo.Addr,
				Err:     serverInfo.Error,
			})
			continue
		}

		// Construct the backend status
		storageInfoStat := backendStatus{}

		for index, mountPath := range storageInfo.MountPaths {
			if foundMatchingPath(mountPath, serverInfo.Addr) {
				storageInfoStat.Used += storageInfo.Used[index]
				storageInfoStat.Available += storageInfo.Available[index]
				storageInfoStat.Total += storageInfo.Total[index]
			}
		}

		if storageInfo.Backend.Type == madmin.Erasure {
			storageInfoStat.Backend = xlBackend{
				Type:             erasureType,
				OnlineDisks:      filterPerNode(serverInfo.Addr, storageInfo.Backend.OnlineDisks),
				OfflineDisks:     filterPerNode(serverInfo.Addr, storageInfo.Backend.OfflineDisks),
				StandardSCData:   storageInfo.Backend.StandardSCData,
				StandardSCParity: storageInfo.Backend.StandardSCParity,
				RRSCData:         storageInfo.Backend.RRSCData,
				RRSCParity:       storageInfo.Backend.RRSCParity,
				Sets:             storageInfo.Backend.Sets,
			}
		} else {
			storageInfoStat.Backend = fsBackend{
				Type: fsType,
			}
		}

		infoMessages = append(infoMessages, (infoMessage{
			Service: "on",
			Addr:    serverInfo.Addr,
			Err:     serverInfo.Error,
			ServerInfo: &ServerInfo{
				ConnStats:  serverInfo.Data.ConnStats,
				Properties: serverInfo.Data.Properties,
				CPULoad:    cpuLoad,
				MemUsage:   memUsage,
			},
			StorageInfo: storageInfoStat,
		}))

	}

	sort.Sort(&sortInfoWrapper{infoMessages})

	for _, s := range infoMessages {
		printMsg(s)
	}

	return nil
}

type sortInfoWrapper struct {
	infos []infoMessage
}

func (s *sortInfoWrapper) Len() int {
	return len(s.infos)
}

func (s *sortInfoWrapper) Swap(i, j int) {
	if s.infos != nil {
		s.infos[i], s.infos[j] = s.infos[j], s.infos[i]
	}
}

func (s *sortInfoWrapper) Less(i, j int) bool {
	if s.infos != nil {
		return s.infos[i].Addr < s.infos[j].Addr
	}
	return false

}
