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
	adminServiceStatusFlags = []cli.Flag{}
)

var adminServiceStatusCmd = cli.Command{
	Name:   "status",
	Usage:  "Get the status of a Minio server",
	Action: mainAdminServiceStatus,
	Before: setGlobalsFromContext,
	Flags:  append(adminServiceStatusFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Get storage information of a Minio server represented by its alias 'play'.
       $ {{.HelpName}} play/

`,
}

// backendType - indicates the type of backend storage
type backendType string

const (
	fsType = backendType("FS")
	xlType = backendType("XL")
)

// fsBackend contains specific FS storage information
type fsBackend struct {
	Type backendType `json:"backendType"`
}

// xlBackend contains specific XL storage information
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

type serverVersion struct {
	Version  string `json:"version"`
	CommitID string `json:"commitID"`
}

// serviceStatusMessage container to hold service status information.
type serviceStatusMessage struct {
	Status        string        `json:"status"`
	Service       bool          `json:"service"`
	StorageInfo   backendStatus `json:"storageInfo"`
	ServerVersion serverVersion `json:"server"`
}

// String colorized service status message.
func (u serviceStatusMessage) String() (msg string) {
	defer func() {
		msg = console.Colorize("ServiceStatus", msg)
	}()
	// When service is offline
	if !u.Service {
		msg = "The server is offline."
		return
	}
	// Online service, get backend information
	msg = fmt.Sprintf("Total: %s, Free: %s.",
		humanize.IBytes(uint64(u.StorageInfo.Total)),
		humanize.IBytes(uint64(u.StorageInfo.Free)),
	)
	if v, ok := u.StorageInfo.Backend.(xlBackend); ok {
		msg += fmt.Sprintf(" Online Disks: %d, Offline Disks: %d.\n", v.OnlineDisks, v.OfflineDisks)
	}
	return
}

// JSON jsonified service status Message message.
func (u serviceStatusMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminServiceStatusSyntax - validate all the passed arguments
func checkAdminServiceStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "status", 1) // last argument is exit code
	}
}

func mainAdminServiceStatus(ctx *cli.Context) error {

	// Validate serivce status syntax.
	checkAdminServiceStatusSyntax(ctx)

	console.SetColor("ServiceStatus", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Fetch the storage info of the specified Minio server
	st, e := client.ServiceStatus()

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
		printMsg(serviceStatusMessage{Service: false})
		return nil
	}

	// If the error is not nil and not unrecognizable, just print it and exit
	fatalIf(probe.NewError(e), "Cannot get service status.")

	// Construct the version response
	version := serverVersion{
		Version:  st.ServerVersion.Version,
		CommitID: st.ServerVersion.CommitID,
	}

	// Construct the backend status
	storageInfo := backendStatus{
		Total: st.StorageInfo.Total,
		Free:  st.StorageInfo.Free,
	}
	if st.StorageInfo.Backend.Type == madmin.XL {
		storageInfo.Backend = xlBackend{
			Type:         xlType,
			OnlineDisks:  st.StorageInfo.Backend.OnlineDisks,
			OfflineDisks: st.StorageInfo.Backend.OfflineDisks,
		}
	} else {
		storageInfo.Backend = fsBackend{
			Type: fsType,
		}
	}

	// Print the whole response
	printMsg(serviceStatusMessage{
		Service:       true,
		StorageInfo:   storageInfo,
		ServerVersion: version,
	})

	return nil
}
