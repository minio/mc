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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var (
	adminLockClearFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "duration, d",
			Usage: "Only clear locks held for longer than NN[h|m|s]",
			Value: "0s",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force a clear lock operation",
		},
	}
)

var adminLockClearCmd = cli.Command{
	Name:   "clear",
	Usage:  "Clear locks held in a given Minio server",
	Before: setGlobalsFromContext,
	Action: mainAdminLockClear,
	Flags:  append(adminLockClearFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Clear all locks held on 'testbucket' in a Minio server with alias 'play'.
       $ {{.HelpName}} --force play/testbucket

    2. Clear all 'testbucket' locks older than 15 minutes.
       $ {{.HelpName}} --force --duration 15m play/testbucket/

    3. Clear all locks held on all objects under prefix 'dir'.
       $ {{.HelpName}} --force play/testbucket/dir/

`,
}

// lockClearMessage container to hold locks information.
type lockClearMessage struct {
	Status string `json:"status"`
	madmin.VolumeLockInfo
}

// String colorized service status message.
func (u lockClearMessage) String() string {
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
func (u lockClearMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminLockClearSyntax - validate all the passed arguments
func checkAdminLockClearSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "clear", 1)
	}

	// Check if a bucket is specified.
	aliasedURL := filepath.ToSlash(ctx.Args().Get(0))
	splits := splitStr(aliasedURL, "/", 3)
	if splits[1] == "" {
		fatalIf(errBucketNotSpecified().Trace(aliasedURL), "Cannot clear locks.")
	}

	if isForce := ctx.Bool("force"); isForce {
		return
	}

	fatalIf(errDummy().Trace(),
		"Clearing locks requires --force flag. This operation is "+
			"*IRREVERSIBLE*. Please review carefully before"+
			" performing this *DANGEROUS* operation.")
}

func mainAdminLockClear(ctx *cli.Context) error {

	checkAdminLockClearSyntax(ctx)

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

	// Clear locks related to a specified pair of bucket and prefix
	locksInfo, e := client.ClearLocks(splits[1], splits[2], duration)
	fatalIf(probe.NewError(e), "Cannot clear the specified locks.")

	for _, l := range locksInfo {
		printMsg(lockClearMessage{VolumeLockInfo: l})
	}

	return nil
}
