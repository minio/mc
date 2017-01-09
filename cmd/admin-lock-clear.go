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
	adminLockClearFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "older-than, o",
			Usage: "Only show locks that are older than a specified time, i.e. 30s",
			Value: "0s",
		},
	}
)

var adminLockClearCmd = cli.Command{
	Name:   "clear",
	Usage:  "Clear locks hold in a given Minio server",
	Action: mainAdminLockClear,
	Flags:  append(adminLockClearFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc admin lock {{.Name}} - {{.Usage}}

USAGE:
   mc admin lock {{.Name}} ALIAS/BUCKET/PREFIX

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}

EXAMPLES:
    1. Clear all locks hold by 'testbucket' in a Minio server represented by its alias 'play'.
       $ mc admin lock {{.Name}} play/testbucket

    2. Clear all 'testbucket' locks that are older than 15 minutes.
       $ mc admin lock {{.Name}} --older-than 15m play/testbucket/

    3. Clear all locks hold by all objects under 'dir' prefix
       $ mc admin lock {{.Name}} play/testbucket/dir/

`,
}

// lockClearMessage container to hold locks information.
type lockClearMessage struct {
	Status   string                `json:"status"`
	LockInfo madmin.VolumeLockInfo `json:"volumeLockInfo"`
}

// String colorized service status message.
func (u lockClearMessage) String() string {
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
func (u lockClearMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminLockClearSyntax - validate all the passed arguments
func checkAdminLockClearSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "clear", 1) // last argument is exit code
	}
}

func mainAdminLockClear(ctx *cli.Context) error {

	setGlobalsFromContext(ctx)
	checkAdminLockClearSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Parse older-than flag
	olderThan, e := time.ParseDuration(ctx.String("older-than"))
	fatalIf(probe.NewError(e), "Unable to parse the passed older-than flag.")

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	aliasedURL = filepath.ToSlash(aliasedURL)

	splits := splitStr(aliasedURL, "/", 3)

	// Clear locks related to a specified pair of bucket and prefix
	locksInfo, e := client.ClearLocks(splits[1], splits[2], olderThan)
	fatalIf(probe.NewError(e), "Cannot clear the specified locks.")

	for _, l := range locksInfo {
		printMsg(lockClearMessage{LockInfo: l})
	}

	return nil
}
