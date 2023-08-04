// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var supportTopLocksFlag = []cli.Flag{
	cli.BoolFlag{
		Name:  "stale",
		Usage: "list stale locks   ask",
	},
	cli.IntFlag{
		Name:   "count",
		Usage:  "number of top locks",
		Hidden: true,
		Value:  10,
	},
}

var supportTopLocksCmd = cli.Command{
	Name:         "locks",
	Usage:        "get a list of the 10 oldest locks on a MinIO cluster.",
	Before:       setGlobalsFromContext,
	Action:       mainSupportTopLocks,
	OnUsageError: onUsageError,
	Flags:        append(supportTopLocksFlag, supportGlobalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get a list of the 10 oldest locks on a MinIO cluster.
     {{.Prompt}} {{.HelpName}} myminio/
`,
}

// lockMessage struct to list lock information.
type lockMessage struct {
	Status string           `json:"status"`
	Lock   madmin.LockEntry `json:"locks"`
}

func getLockDuration(duration time.Duration) (string, string) {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
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
	const (
		timeFieldMaxLen     = 20
		resourceFieldMaxLen = -1
		typeFieldMaxLen     = 6
	)

	elapsed := u.Lock.Elapsed
	// elapsed can be zero with older MinIO versions,
	// so this code is deprecated and can be removed later.
	if elapsed == 0 {
		elapsed = time.Now().UTC().Sub(u.Lock.Timestamp)
	}

	lockState, timeDiff := getLockDuration(elapsed)
	return console.Colorize(lockState, newPrettyTable("  ",
		Field{"Time", timeFieldMaxLen},
		Field{"Type", typeFieldMaxLen},
		Field{"Resource", resourceFieldMaxLen},
	).buildRow(timeDiff, u.Lock.Type, u.Lock.Resource))
}

// JSON jsonified top oldest locks message.
func (u lockMessage) JSON() string {
	type lockEntry struct {
		Timestamp  time.Time `json:"time"`       // When the lock was first granted
		Elapsed    string    `json:"elapsed"`    // Humanized duration for which lock has been held
		Resource   string    `json:"resource"`   // Resource contains info like bucket+object
		Type       string    `json:"type"`       // Type indicates if 'Write' or 'Read' lock
		Source     string    `json:"source"`     // Source at which lock was granted
		ServerList []string  `json:"serverlist"` // List of servers participating in the lock.
		Owner      string    `json:"owner"`      // Owner UUID indicates server owns the lock.
		ID         string    `json:"id"`         // UID to uniquely identify request of client.
		// Represents quorum number of servers required to hold this lock, used to look for stale locks.
		Quorum int `json:"quorum"`
	}

	le := lockEntry{
		Timestamp:  u.Lock.Timestamp,
		Elapsed:    u.Lock.Elapsed.Round(time.Second).String(),
		Resource:   u.Lock.Resource,
		Type:       u.Lock.Type,
		Source:     u.Lock.Source,
		ServerList: u.Lock.ServerList,
		Owner:      u.Lock.Owner,
		ID:         u.Lock.ID,
		Quorum:     u.Lock.Quorum,
	}
	statusJSONBytes, e := json.MarshalIndent(le, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(statusJSONBytes)
}

// checkAdminTopLocksSyntax - validate all the passed arguments
func checkSupportTopLocksSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainSupportTopLocks(ctx *cli.Context) error {
	checkSupportTopLocksSyntax(ctx)
	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)
	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call top locks API
	entries, e := client.TopLocksWithOpts(globalContext, madmin.TopLockOpts{
		Count: ctx.Int("count"),
		Stale: ctx.Bool("stale"),
	})
	fatalIf(probe.NewError(e), "Unable to get server locks list.")

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
	console.Println(console.Colorize("Headers", newPrettyTable("  ",
		Field{"Time", timeFieldMaxLen},
		Field{"Type", typeFieldMaxLen},
		Field{"Resource", resourceFieldMaxLen},
	).buildRow("Time", "Type", "Resource")))
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
