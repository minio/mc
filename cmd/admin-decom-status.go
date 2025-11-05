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
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminDecommissionStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "show current decommissioning status",
	Action:       mainAdminDecommissionStatus,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show current decommissioning status.
     {{.Prompt}} {{.HelpName}} myminio/ http://server{5...8}/disk{1...4}
  2. List all current decommissioning status of all pools.
     {{.Prompt}} {{.HelpName}} myminio/
`,
}

// checkAdminDecommissionStatusSyntax - validate all the passed arguments
func checkAdminDecommissionStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 2 || len(ctx.Args()) == 0 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainAdminDecommissionStatus is the handle for "mc admin decomission status" command.
func mainAdminDecommissionStatus(ctx *cli.Context) error {
	checkAdminDecommissionStatusSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	aliasedURL = filepath.Clean(aliasedURL)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if pool := args.Get(1); pool != "" {
		poolStatus, e := client.StatusPool(globalContext, pool)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get status per pool")

		if globalJSON {
			statusJSONBytes, e := json.MarshalIndent(poolStatus, "", "    ")
			fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
			console.Println(string(statusJSONBytes))
			return nil
		}

		var msg string
		if poolStatus.Decommission.Complete {
			msg = color.GreenString(fmt.Sprintf("Decommission of pool %s is complete, you may now remove it from server command line", poolStatus.CmdLine))
		} else if poolStatus.Decommission.Failed {
			msg = color.GreenString(fmt.Sprintf("Decommission of pool %s failed, please retry again", poolStatus.CmdLine))
		} else if poolStatus.Decommission.Canceled {
			msg = color.GreenString(fmt.Sprintf("Decommission of pool %s was canceled, you may start again", poolStatus.CmdLine))
		} else if !poolStatus.Decommission.StartTime.IsZero() {
			usedStart := (poolStatus.Decommission.TotalSize - poolStatus.Decommission.StartSize)
			usedCurrent := (poolStatus.Decommission.TotalSize - poolStatus.Decommission.CurrentSize)

			duration := float64(time.Since(poolStatus.Decommission.StartTime)) / float64(time.Second)
			if usedStart > usedCurrent && duration > 10 {
				copied := uint64(usedStart - usedCurrent)
				speed := uint64(float64(copied) / duration)
				msg = "Decommissioning rate at " + humanize.IBytes(speed) + "/sec " + "[" + humanize.IBytes(
					uint64(usedCurrent)) + "/" + humanize.IBytes(uint64(poolStatus.Decommission.TotalSize)) + "]"
				msg += "\nStarted: " + humanize.RelTime(time.Now().UTC(), poolStatus.Decommission.StartTime, "", "ago")
			} else {
				msg = "Decommissioning is starting..."
			}
			msg = color.GreenString(msg)
		} else {
			errorIf(errDummy().Trace(args...), "This pool is currently not scheduled for decomissioning")
			return nil
		}
		fmt.Println(msg)
		return nil
	}
	poolStatuses, e := client.ListPoolsStatus(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get status for all pools")

	if globalJSON {
		statusJSONBytes, e := json.MarshalIndent(poolStatuses, "", "    ")
		fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
		console.Println(string(statusJSONBytes))
		return nil
	}

	dspOrder := []col{colGreen} // Header
	for range poolStatuses {
		dspOrder = append(dspOrder, colGrey)
	}
	var printColors []*color.Color
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}

	tbl := console.NewTable(printColors, []bool{false, false, false, false}, 0)

	cellText := make([][]string, len(poolStatuses)+1)
	cellText[0] = []string{
		"ID",
		"Pools",
		"Drives Usage",
		"Status",
	}
	for idx, pool := range poolStatuses {
		idx++
		totalSize := uint64(pool.Decommission.TotalSize)
		usedCurrent := uint64(pool.Decommission.TotalSize - pool.Decommission.CurrentSize)
		var capacity string
		if totalSize == 0 {
			capacity = "0% (total: 0B)"
		} else {
			capacity = fmt.Sprintf("%.1f%% (total: %s)", 100*float64(usedCurrent)/float64(totalSize), humanize.IBytes(totalSize))
		}
		status := "Active"
		if pool.Decommission != nil {
			if pool.Decommission.Complete {
				status = "Complete"
			} else if pool.Decommission.Failed {
				status = "Draining(Failed)"
			} else if pool.Decommission.Canceled {
				status = "Draining(Canceled)"
			} else if !pool.Decommission.StartTime.IsZero() {
				status = "Draining"
			}
		}
		cellText[idx] = []string{
			humanize.Ordinal(pool.ID + 1),
			pool.CmdLine,
			capacity,
			status,
		}
	}
	return tbl.DisplayTable(cellText)
}
