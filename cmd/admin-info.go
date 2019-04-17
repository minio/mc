/*
 * MinIO Client (C) 2016, 2017, 2018 MinIO, Inc.
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

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var (
	adminInfoFlags = []cli.Flag{}
)

var adminInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "display MinIO server information",
	Action: mainAdminInfo,
	Before: setGlobalsFromContext,
	Flags:  append(adminInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server information of the 'play' MinIO server.
       $ {{.HelpName}} play/

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
	Used    uint64      `json:"used"`
	Backend interface{} `json:"backend"`
}

// ServerInfo holds the whole server information that will be
// returned by ServerInfo API.
type ServerInfo struct {
	StorageInfo backendStatus           `json:"storage"`
	ConnStats   madmin.ServerConnStats  `json:"network"`
	Properties  madmin.ServerProperties `json:"server"`
}

// infoMessage container to hold service status information.
type infoMessage struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Addr    string `json:"address"`
	Err     string `json:"error"`
	*ServerInfo
}

// String colorized service status message.
func (u infoMessage) String() (msg string) {
	defer func() {
		msg += "\n"
	}()

	dot := "●"

	// When service is offline
	if u.Service == "off" {
		msg += fmt.Sprintf("%s  %s\n", dot, console.Colorize("PrintB", u.Addr))
		msg += fmt.Sprintf("  Status : %s", console.Colorize("InfoFail", "offline"))
		return
	}

	// Print error if any and exit
	if u.Err != "" {
		msg += fmt.Sprintf("%s  %s\n", dot, console.Colorize("PrintB", u.Addr))
		msg += fmt.Sprintf("   Status : %s\n", console.Colorize("InfoFail", "offline"))
		e := u.Err
		if strings.Trim(e, " ") == "rpc: retry error" {
			e = "unreachable"
		}
		msg += fmt.Sprintf("    Error : %s", console.Colorize("InfoFail", e))
		return
	}

	// Print server title
	msg += fmt.Sprintf("%s  %s\n", dot, console.Colorize("PrintB", u.Addr))

	// Status
	msg += fmt.Sprintf("   Status : %s\n", console.Colorize("Info", "online"))

	// Uptime
	msg += fmt.Sprintf("   Uptime : %s\n", humanize.RelTime(time.Now(), time.Now().Add(-u.ServerInfo.Properties.Uptime), "", ""))
	// Version
	version := u.ServerInfo.Properties.Version
	if u.ServerInfo.Properties.Version == "DEVELOPMENT.GOGET" {
		version = "<development>"
	}
	msg += fmt.Sprintf("  Version : %s\n", version)
	// Region
	if u.ServerInfo.Properties.Region != "" {
		msg += fmt.Sprintf("   Region : %s\n", u.ServerInfo.Properties.Region)
	}
	// ARNs
	sqsARNs := ""
	for _, v := range u.ServerInfo.Properties.SQSARN {
		sqsARNs += fmt.Sprintf("%s ", v)
	}
	if sqsARNs != "" {
		msg += fmt.Sprintf(" SQS ARNs : %s\n", sqsARNs)
	}

	// Incoming/outgoing
	msg += fmt.Sprintf("  Storage : Used %s", humanize.IBytes(u.StorageInfo.Used))
	if v, ok := u.ServerInfo.StorageInfo.Backend.(xlBackend); ok {
		upBackends := 0
		downBackends := 0
		for _, s := range v.Sets {
			for _, s := range s {
				if strings.Contains(s.Endpoint, u.Addr) {
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
		msg += fmt.Sprintf("\n   Drives : %s/%d OK\n", upBackendsString, upBackends+downBackends)
	}
	return
}

// JSON jsonified service status Message message.
func (u infoMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate all the passed arguments
func checkAdminInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}
}

func mainAdminInfo(ctx *cli.Context) error {
	// Validate service status syntax.
	checkAdminInfoSyntax(ctx)

	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoDegraded", color.New(color.FgYellow, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Fetch info of all servers (cluster or single server)
	serversInfo, e := client.ServerInfo()

	// Check the availability of the server: online or offline. A server is considered
	// offline if we can't get any response or we get a bad format response
	var serviceOffline bool
	switch e.(type) {
	case *json.SyntaxError:
		serviceOffline = true
	case *url.Error:
		serviceOffline = true
	}

	if serviceOffline {
		printMsg(infoMessage{Addr: aliasedURL, Service: "off"})
		return nil
	}

	// If the error is not nil and not unrecognizable, just print it and exit
	fatalIf(probe.NewError(e), "Cannot get service status.")

	infoMessages := []infoMessage{}
	for _, serverInfo := range serversInfo {
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
		storageInfo := backendStatus{
			Used: serverInfo.Data.StorageInfo.Used,
		}

		if serverInfo.Data.StorageInfo.Backend.Type == madmin.Erasure {
			storageInfo.Backend = xlBackend{
				Type:             erasureType,
				OnlineDisks:      serverInfo.Data.StorageInfo.Backend.OnlineDisks,
				OfflineDisks:     serverInfo.Data.StorageInfo.Backend.OfflineDisks,
				StandardSCData:   serverInfo.Data.StorageInfo.Backend.StandardSCData,
				StandardSCParity: serverInfo.Data.StorageInfo.Backend.StandardSCParity,
				RRSCData:         serverInfo.Data.StorageInfo.Backend.RRSCData,
				RRSCParity:       serverInfo.Data.StorageInfo.Backend.RRSCParity,
				Sets:             serverInfo.Data.StorageInfo.Backend.Sets,
			}
		} else {
			storageInfo.Backend = fsBackend{
				Type: fsType,
			}
		}

		infoMessages = append(infoMessages, (infoMessage{
			Service: "on",
			Addr:    serverInfo.Addr,
			Err:     serverInfo.Error,
			ServerInfo: &ServerInfo{
				StorageInfo: storageInfo,
				ConnStats:   serverInfo.Data.ConnStats,
				Properties:  serverInfo.Data.Properties,
			},
		}))

	}

	sort.Stable(&sortInfoWrapper{infoMessages})
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
