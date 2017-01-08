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
	"path/filepath"
	"time"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/probe"
)

var (
	adminLockListFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "older-than, o",
			Usage: "Only show locks that are older than NN[h|m|s]. (default: \"24h\")",
			Value: "24h",
		},
	}
)

var adminLockListCmd = cli.Command{
	Name:   "list",
	Usage:  "Get the list of locks hold in a given Minio server",
	Action: mainAdminLockList,
	Flags:  append(adminLockListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc admin lock {{.Name}} - {{.Usage}}

USAGE:
   mc admin lock {{.Name}} ALIAS/BUCKET/PREFIX

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}

EXAMPLES:
    1. List hold locks related to testbucket in a Minio server represented by its alias 'play'.
       $ mc admin lock {{.Name}} play/testbucket/

    2. List locks that are hold for more than 15 minutes.
       $ mc admin lock {{.Name}} --older-than 15m play/testbucket/

    3. List locks hold by all objects under dir prefix
       $ mc admin lock {{.Name}} play/testbucket/dir/

`,
}

// lockListMessage container to hold locks information.
type lockListMessage struct {
	Status   string                `json:"status"`
	LockInfo madmin.VolumeLockInfo `json:"volumeLockInfo"`
}

// String colorized service status message.
func (u lockListMessage) String() string {
	msg := fmt.Sprintf("%s/%s (LocksOnObject: %d, locksAcquiredOnObject: %d, totalBlockLocks:%d): ",
		u.LockInfo.Bucket,
		u.LockInfo.Object,
		u.LockInfo.LocksOnObject,
		u.LockInfo.LocksAcquiredOnObject,
		u.LockInfo.TotalBlockedLocks)
	for _, detail := range u.LockInfo.LockDetailsOnObject {
		msg += fmt.Sprintf("  %+v", detail)
	}
	msg += "\n"
	return msg
}

// JSON jsonified service status Message message.
func (u lockListMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminLockListSyntax - validate all the passed arguments
func checkAdminLockListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

func mainAdminLockList(ctx *cli.Context) error {

	setGlobalsFromContext(ctx)
	checkAdminLockListSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Parse older-than flag
	olderThan, e := time.ParseDuration(ctx.String("older-than"))
	fatalIf(probe.NewError(e), "Unable to parse the passed older-than flag.")

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		return err.ToGoError()
	}

	aliasedURL = filepath.ToSlash(aliasedURL)

	splits := splitStr(aliasedURL, "/", 3)

	// Fetch the lock info related to a specified pair of bucket and prefix
	locksInfo, e := client.ListLocks(splits[1], splits[2], olderThan)
	fatalIf(probe.NewError(e), "Cannot get lock status.")

	for _, l := range locksInfo {
		printMsg(lockListMessage{LockInfo: l})
	}

	return nil
}
