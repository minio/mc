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
			Name:  "duration, d",
			Usage: "Only show locks that are hold for longer than NN[h|m|s].",
			Value: "24h",
		},
	}
)

var adminLockListCmd = cli.Command{
	Name:   "list",
	Usage:  "Get the list of locks hold in a given Minio server",
	Action: mainAdminLockList,
	Before: setGlobalsFromContext,
	Flags:  append(adminLockListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} ALIAS/BUCKET/PREFIX

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. List hold locks related to testbucket in a Minio server represented by its alias 'play'.
       $ {{.HelpName}} play/testbucket/

    2. List locks that are hold for more than 15 minutes.
       $ {{.HelpName}} --duration 15m play/testbucket/

    3. List locks hold by all objects under dir prefix
       $ {{.HelpName}} play/testbucket/dir/

`,
}

// lockListMessage container to hold locks information.
type lockListMessage struct {
	Status string `json:"status"`
	madmin.VolumeLockInfo
}

// String colorized service status message.
func (u lockListMessage) String() string {
	msg := fmt.Sprintf("%s/%s (LocksOnObject: %d, locksAcquiredOnObject: %d, totalBlockLocks:%d): ",
		u.Bucket,
		u.Object,
		u.LocksOnObject,
		u.LocksAcquiredOnObject,
		u.TotalBlockedLocks)
	for _, detail := range u.LockDetailsOnObject {
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

	checkAdminLockListSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Parse duration flag
	duration, e := time.ParseDuration(ctx.String("duration"))
	fatalIf(probe.NewError(e), "Unable to parse the passed duration flag.")

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	aliasedURL = filepath.ToSlash(aliasedURL)

	splits := splitStr(aliasedURL, "/", 3)

	// Fetch the lock info related to a specified pair of bucket and prefix
	locksInfo, e := client.ListLocks(splits[1], splits[2], duration)
	fatalIf(probe.NewError(e), "Cannot get lock status.")

	for _, l := range locksInfo {
		printMsg(lockListMessage{VolumeLockInfo: l})
	}

	return nil
}
