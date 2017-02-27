/*
 * Minio Client (C) 2016 Minio, Inc.
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

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/probe"
)

var (
	adminInfoFlags = []cli.Flag{}
)

var adminInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "Get information of a Minio server",
	Action: mainAdminInfo,
	Before: setGlobalsFromContext,
	Flags:  append(adminInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server information of the 'play' Minio server.
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
}

// backendStatus represents the overall information of all backend storage types
type backendStatus struct {
	Total   int64       `json:"total"`
	Free    int64       `json:"free"`
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
	Service bool   `json:"service"`
	*ServerInfo
}

// String colorized service status message.
func (u infoMessage) String() (msg string) {
	defer func() {
		msg = console.Colorize("Info", msg)
	}()
	// When service is offline
	if !u.Service {
		msg = "The server is offline."
		return
	}
	msg += fmt.Sprintf(" Version : %s\n", u.ServerInfo.Properties.Version)
	msg += fmt.Sprintf("  Uptime : %s\n", timeDurationToHumanizedDuration(u.ServerInfo.Properties.Uptime))
	msg += fmt.Sprintf("  Region : %s\n", u.ServerInfo.Properties.Region)
	msg += fmt.Sprintf(" Network : Incoming %s, Outgoing %s\n",
		humanize.IBytes(u.ServerInfo.ConnStats.TotalInputBytes),
		humanize.IBytes(u.ServerInfo.ConnStats.TotalOutputBytes))

	// Online service, get backend information
	msg += fmt.Sprintf(" Storage : Total %s, Free %s",
		humanize.IBytes(uint64(u.StorageInfo.Total)),
		humanize.IBytes(uint64(u.StorageInfo.Free)),
	)
	if v, ok := u.ServerInfo.StorageInfo.Backend.(xlBackend); ok {
		msg += fmt.Sprintf(", Online Disks: %d, Offline Disks: %d\n", v.OnlineDisks, v.OfflineDisks)
	}
	return
}

// JSON jsonified service status Message message.
func (u infoMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
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

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Fetch the server info of the specified Minio server
	serverInfo, e := client.ServerInfo()

	// Check the availability of the server: online or offline. A server is considered
	// offline if we can't get any response or we get a bad format response
	var serviceOffline bool
	switch v := e.(type) {
	case *json.SyntaxError:
		serviceOffline = true
	case *url.Error:
		if v.Timeout() {
			serviceOffline = true
		}
	}
	if serviceOffline {
		printMsg(infoMessage{Service: false})
		return nil
	}

	// If the error is not nil and not unrecognizable, just print it and exit
	fatalIf(probe.NewError(e), "Cannot get service status.")

	// Construct the backend status
	storageInfo := backendStatus{
		Total: serverInfo.StorageInfo.Total,
		Free:  serverInfo.StorageInfo.Free,
	}
	if serverInfo.StorageInfo.Backend.Type == madmin.Erasure {
		storageInfo.Backend = xlBackend{
			Type:         erasureType,
			OnlineDisks:  serverInfo.StorageInfo.Backend.OnlineDisks,
			OfflineDisks: serverInfo.StorageInfo.Backend.OfflineDisks,
		}
	} else {
		storageInfo.Backend = fsBackend{
			Type: fsType,
		}
	}

	printMsg(infoMessage{
		Service: true,
		ServerInfo: &ServerInfo{
			StorageInfo: storageInfo,
			ConnStats:   serverInfo.ConnStats,
			Properties:  serverInfo.Properties,
		},
	})

	return nil
}
