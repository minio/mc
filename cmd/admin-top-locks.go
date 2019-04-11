/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminTopLocksCmd = cli.Command{
	Name:   "locks",
	Usage:  "Get a list of the 10 oldest locks on a MinIO cluster.",
	Before: setGlobalsFromContext,
	Action: mainAdminTopLocks,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get a list of the 10 oldest locks on a MinIO cluster.
     $ {{.HelpName}} myminio/

`,
}

// lockMessage struct to list lock information.
type lockMessage struct {
	Status string           `json:"status"`
	Lock   madmin.LockEntry `json:"locks"`
}

func getTimeDiff(timeStamp time.Time) (string, string) {
	now := time.Now().UTC()
	diff := now.Sub(timeStamp)
	hours := int(diff.Hours())
	minutes := int(diff.Minutes()) % 60
	seconds := int(diff.Seconds()) % 60
	if hours == 0 {
		if minutes == 0 {
			return "Lock", fmt.Sprint(seconds, " seconds")
		}
		return "Lock", fmt.Sprint(minutes, " minutes")
	}
	return "StaleLock", fmt.Sprint(hours, " hours")
}

// String colorized oldest locks message.
func (u lockMessage) String() string {
	timeFieldMaxLen := 20
	resourceFieldMaxLen := -1
	typeFieldMaxLen := 6
	ownerFieldMaxLen := 20

	lockState, timeDiff := getTimeDiff(u.Lock.Timestamp)
	return console.Colorize(lockState, newPrettyTable("  ",
		Field{"Time", timeFieldMaxLen},
		Field{"Type", typeFieldMaxLen},
		Field{"Owner", ownerFieldMaxLen},
		Field{"Resource", resourceFieldMaxLen},
	).buildRow(timeDiff, u.Lock.Type, u.Lock.Owner, u.Lock.Resource))
}

// JSON jsonified top oldest locks message.
func (u lockMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(statusJSONBytes)
}

// checkAdminTopLocksSyntax - validate all the passed arguments
func checkAdminTopLocksSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "locks", 1) // last argument is exit code
	}
}

func mainAdminTopLocks(ctx *cli.Context) error {

	checkAdminTopLocksSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Call top locks API
	entries, e := client.TopLocks()
	fatalIf(probe.NewError(e), "Cannot get server locks list.")

	console.SetColor("StaleLock", color.New(color.FgRed, color.Bold))
	console.SetColor("Lock", color.New(color.FgBlue, color.Bold))
	console.SetColor("Headers", color.New(color.FgGreen, color.Bold))

	// Print
	printLocks(entries)
	return nil
}

func printHeaders() {
	timeFieldMaxLen := 20
	resourceFieldMaxLen := -1
	typeFieldMaxLen := 6
	ownerFieldMaxLen := 20
	console.Println(console.Colorize("Headers", newPrettyTable("  ",
		Field{"Time", timeFieldMaxLen},
		Field{"Type", typeFieldMaxLen},
		Field{"Owner", ownerFieldMaxLen},
		Field{"Resource", resourceFieldMaxLen},
	).buildRow("Time", "Type", "Owner", "Resource")))
}

// Prints oldest locks.
func printLocks(locks madmin.LockEntries) {
	if !globalJSON {
		printHeaders()
	}
	for _, entry := range locks {
		printMsg(lockMessage{Lock: entry})
	}
}
