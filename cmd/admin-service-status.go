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

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
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
	Flags:  append(adminServiceStatusFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc admin service {{.Name}} - {{.Usage}}

USAGE:
   mc admin service {{.Name}} ALIAS

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
    1. Get storage information of a Minio server represented by its alias 'play'.
       $ mc admin service {{.Name}} play/
`,
}

// serviceStatusMessage container to hold service status information.
type serviceStatusMessage struct {
	Status      string                       `json:"status"`
	StorageInfo madmin.ServiceStatusMetadata `json:"storageInfo"`
}

// String colorized service status message.
func (u serviceStatusMessage) String() string {
	msg := fmt.Sprintf("Total: %s, Free: %s.",
		humanize.IBytes(uint64(u.StorageInfo.Total)),
		humanize.IBytes(uint64(u.StorageInfo.Free)),
	)
	if u.StorageInfo.Backend.Type == madmin.XL {
		msg += fmt.Sprintf(" Online Disks: %d, Offline Disks: %d\n",
			u.StorageInfo.Backend.OnlineDisks,
			u.StorageInfo.Backend.OfflineDisks)
	}
	return msg
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

	setGlobalsFromContext(ctx)
	checkAdminServiceStatusSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Fetch the storage info of the specified Minio server
	status, e := client.ServiceStatus()
	fatalIf(probe.NewError(e), "Cannot get service status.")

	printMsg(serviceStatusMessage{StorageInfo: status})

	return nil
}
