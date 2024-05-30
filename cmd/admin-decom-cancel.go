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
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminDecommissionCancelCmd = cli.Command{
	Name:         "cancel",
	Usage:        "cancel an ongoing decommissioning of a pool",
	Action:       mainAdminDecommissionCancel,
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
  1. Cancel an ongoing decommissioning of a pool.
     {{.Prompt}} {{.HelpName}} myminio/ http://server{5...8}/disk{1...4}

  2. Cancel all ongoing decommissioning of pools.
     {{.Prompt}} {{.HelpName}} myminio/
`,
}

// checkAdminDecommissionCancelSyntax - validate all the passed arguments
func checkAdminDecommissionCancelSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 2 || len(ctx.Args()) == 0 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainAdminDecommissionCancel is the handle for "mc admin decommission cancel" command.
func mainAdminDecommissionCancel(ctx *cli.Context) error {
	checkAdminDecommissionCancelSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	aliasedURL = filepath.Clean(aliasedURL)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if pool := args.Get(1); pool != "" {
		e := client.CancelDecommissionPool(globalContext, pool)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to cancel decommissioning, please try again")
		return nil
	}

	poolStatuses, e := client.ListPoolsStatus(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get status for all pools")

	var newPoolStatuses []madmin.PoolStatus
	for _, pool := range poolStatuses {
		if pool.Decommission != nil {
			if pool.Decommission.StartTime.IsZero() {
				continue
			}
			if pool.Decommission.Complete {
				continue
			}
		}
		newPoolStatuses = append(newPoolStatuses, pool)
	}

	dspOrder := []col{colGreen} // Header
	for i := 0; i < len(newPoolStatuses); i++ {
		dspOrder = append(dspOrder, colGrey)
	}
	var printColors []*color.Color
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}

	tbl := console.NewTable(printColors, []bool{false, false, false, false}, 0)

	cellText := make([][]string, len(newPoolStatuses)+1)
	cellText[0] = []string{
		"ID",
		"Pools",
		"Capacity",
		"Status",
	}
	for idx, pool := range poolStatuses {
		idx++
		totalSize := uint64(pool.Decommission.TotalSize)
		currentSize := uint64(pool.Decommission.CurrentSize)
		capacity := humanize.IBytes(totalSize-currentSize) + " (used) / " + humanize.IBytes(totalSize) + " (total)"
		status := ""
		if pool.Decommission != nil {
			if pool.Decommission.StartTime.IsZero() {
				continue
			}
			if pool.Decommission.Complete {
				continue
			}
			status = "Draining"
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
