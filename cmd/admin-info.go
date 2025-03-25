// Copyright (c) 2015-2024 MinIO, Inc.
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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/v3/console"
)

var adminInfoFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "offline",
		Usage: "show only offline nodes/drives",
	},
}

var adminInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display MinIO server information",
	Action:       mainAdminInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminInfoFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server information of the 'play' MinIO server.
     {{.Prompt}} {{.HelpName}} play/
`,
}

type poolSummary struct {
	index                  int
	setsCount              int
	drivesPerSet           int
	driveTolerance         int
	drivesTotalFreeSpace   uint64
	drivesTotalUsableSpace uint64
	endpoints              set.StringSet
}

type clusterInfo map[int]*poolSummary

func clusterSummaryInfo(info madmin.InfoMessage) clusterInfo {
	summary := make(clusterInfo)

	for _, srv := range info.Servers {
		for _, disk := range srv.Disks {
			if disk.PoolIndex < 0 {
				continue
			}

			pool := summary[disk.PoolIndex]
			if pool == nil {
				pool = &poolSummary{
					index:          disk.PoolIndex,
					endpoints:      set.NewStringSet(),
					driveTolerance: info.StandardParity(),
				}
			}

			if len(info.Backend.DrivesPerSet) > 0 {
				if disk.DiskIndex < (info.Backend.DrivesPerSet[disk.PoolIndex] - info.Backend.StandardSCParity) {
					pool.drivesTotalFreeSpace += disk.AvailableSpace
					pool.drivesTotalUsableSpace += disk.TotalSpace
				}
			}

			pool.endpoints.Add(srv.Endpoint)
			summary[disk.PoolIndex] = pool
		}
	}

	for idx := range info.Backend.TotalSets {
		pool := summary[idx]
		if pool != nil {
			pool.setsCount = info.Backend.TotalSets[idx]
			pool.drivesPerSet = info.Backend.DrivesPerSet[idx]
			summary[idx] = pool
		}
	}

	return summary
}

func endpointToPools(endpoint string, c clusterInfo) (pools []int) {
	for poolNumber, poolSummary := range c {
		if poolSummary.endpoints.Contains(endpoint) {
			pools = append(pools, poolNumber)
		}
	}
	sort.Ints(pools)
	return
}

// Wrap "Info" message together with fields "Status" and "Error"
type clusterStruct struct {
	Status string             `json:"status"`
	Error  string             `json:"error,omitempty"`
	Info   madmin.InfoMessage `json:"info,omitempty"`

	onlyOffline bool
}

// String provides colorized info messages
func (u clusterStruct) String() (msg string) {
	// Check cluster level "Status" field for error
	if u.Status == "error" {
		fatal(probe.NewError(errors.New(u.Error)), "Unable to get service info")
	}

	// If nothing has been collected, error out
	if u.Info.Servers == nil {
		fatal(probe.NewError(errors.New("Unable to get service info")), "")
	}

	// Initialization
	var totalOfflineNodes int

	// Color palette initialization
	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))
	console.SetColor("InfoWarning", color.New(color.FgYellow, color.Bold))

	backendType := u.Info.BackendType()

	coloredDot := console.Colorize("Info", dot)
	if madmin.ItemState(u.Info.Mode) == madmin.ItemInitializing {
		coloredDot = console.Colorize("InfoWarning", dot)
	}

	sort.Slice(u.Info.Servers, func(i, j int) bool {
		return u.Info.Servers[i].Endpoint < u.Info.Servers[j].Endpoint
	})

	clusterSummary := clusterSummaryInfo(u.Info)

	// Loop through each server and put together info for each one
	for _, srv := range u.Info.Servers {
		// Check if MinIO server is not online ("Mode" field),
		if srv.State != string(madmin.ItemOnline) {
			totalOfflineNodes++
			// "PrintB" is color blue in console library package
			msg += fmt.Sprintf("%s  %s\n", console.Colorize("InfoFail", dot), console.Colorize("PrintB", srv.Endpoint))
			msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("InfoFail", srv.State))

			if backendType == madmin.Erasure {
				// Info about drives on a server, only available for non-FS types
				var OffDrives int
				var OnDrives int
				var dispNoOfDrives string
				for _, disk := range srv.Disks {
					switch disk.State {
					case madmin.DriveStateOk, madmin.DriveStateUnformatted:
						OnDrives++
					default:
						OffDrives++
					}
				}

				totalDrivesPerServer := OnDrives + OffDrives

				dispNoOfDrives = strconv.Itoa(OnDrives) + "/" + strconv.Itoa(totalDrivesPerServer)
				msg += fmt.Sprintf("   Drives: %s %s\n", dispNoOfDrives, console.Colorize("InfoFail", "OK "))
			}

			msg += "\n"

			// Continue to the next server
			continue
		}

		if u.onlyOffline {
			continue
		}

		// Print server title
		msg += fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("PrintB", srv.Endpoint))

		// Uptime
		msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("Info",
			humanize.RelTime(time.Now(), time.Now().Add(time.Duration(srv.Uptime)*time.Second), "", "")))

		// Version
		version := srv.Version
		if strings.Contains(srv.Version, "DEVELOPMENT") {
			version = "<development>"
		}
		msg += fmt.Sprintf("   Version: %s\n", version)
		// Network info, only available for non-FS types
		connectionAlive := 0
		totalNodes := len(srv.Network)
		if srv.Network != nil && backendType == madmin.Erasure {
			for _, v := range srv.Network {
				if v == "online" {
					connectionAlive++
				}
			}
			clr := "Info"
			if connectionAlive != totalNodes {
				clr = "InfoWarning"
			}
			displayNwInfo := strconv.Itoa(connectionAlive) + "/" + strconv.Itoa(totalNodes)
			msg += fmt.Sprintf("   Network: %s %s\n", displayNwInfo, console.Colorize(clr, "OK "))
		}

		if backendType == madmin.Erasure {
			// Info about drives on a server, only available for non-FS types
			var OffDrives int
			var OnDrives int
			var dispNoOfDrives string
			for _, disk := range srv.Disks {
				switch disk.State {
				case madmin.DriveStateOk, madmin.DriveStateUnformatted:
					OnDrives++
				default:
					OffDrives++
				}
			}

			totalDrivesPerServer := OnDrives + OffDrives
			clr := "Info"
			if OnDrives != totalDrivesPerServer {
				clr = "InfoWarning"
			}
			dispNoOfDrives = strconv.Itoa(OnDrives) + "/" + strconv.Itoa(totalDrivesPerServer)
			msg += fmt.Sprintf("   Drives: %s %s\n", dispNoOfDrives, console.Colorize(clr, "OK "))

			// Print pools belonging to this server
			var prettyPools []string
			for _, pool := range endpointToPools(srv.Endpoint, clusterSummary) {
				prettyPools = append(prettyPools, strconv.Itoa(pool+1))
			}
			msg += fmt.Sprintf("   Pool: %s\n", console.Colorize("Info", fmt.Sprintf("%+v", strings.Join(prettyPools, ", "))))
		}

		msg += "\n"
	}

	if backendType == madmin.Erasure {
		dspOrder := []col{colGreen} // Header
		for i := 0; i < len(clusterSummary); i++ {
			dspOrder = append(dspOrder, colGrey)
		}
		var printColors []*color.Color
		for _, c := range dspOrder {
			printColors = append(printColors, getPrintCol(c))
		}

		tbl := console.NewTable(printColors, []bool{false, false, false, false}, 0)

		var builder strings.Builder
		cellText := make([][]string, 0, len(clusterSummary)+1)
		cellText = append(cellText, []string{
			"Pool",
			"Drives Usage",
			"Erasure stripe size",
			"Erasure sets",
		})

		var printSummary bool
		// Keep the pool order while printing the output
		for poolIdx := 0; poolIdx < len(clusterSummary); poolIdx++ {
			summary := clusterSummary[poolIdx]
			if summary == nil {
				break
			}
			totalSize := summary.drivesTotalUsableSpace
			usedCurrent := summary.drivesTotalUsableSpace - summary.drivesTotalFreeSpace
			var capacity string
			if totalSize > 0 {
				capacity = fmt.Sprintf("%.1f%% (total: %s)", 100*float64(usedCurrent)/float64(totalSize), humanize.IBytes(totalSize))
			}

			if summary.drivesPerSet > 0 {
				printSummary = true
			}

			cellText = append(cellText, []string{
				humanize.Ordinal(poolIdx + 1),
				capacity,
				strconv.Itoa(summary.drivesPerSet),
				strconv.Itoa(summary.setsCount),
			})
		}

		if printSummary {
			e := tbl.PopulateTable(&builder, cellText)
			fatalIf(probe.NewError(e), "unable to populate the table")

			msg += builder.String() + "\n"
		}
	}

	// Summary on used space, total no of buckets and
	// total no of objects at the Cluster level
	usedTotal := humanize.IBytes(u.Info.Usage.Size)
	if u.Info.Buckets.Count > 0 {
		msg += fmt.Sprintf("%s Used, %s, %s", usedTotal,
			english.Plural(int(u.Info.Buckets.Count), "Bucket", ""),
			english.Plural(int(u.Info.Objects.Count), "Object", ""))
		if u.Info.Versions.Count > 0 {
			msg += ", " + english.Plural(int(u.Info.Versions.Count), "Version", "")
		}
		if u.Info.DeleteMarkers.Count > 0 {
			msg += ", " + english.Plural(int(u.Info.DeleteMarkers.Count), "Delete Marker", "")
		}
		msg += "\n"
	}
	if backendType == madmin.Erasure {
		if totalOfflineNodes != 0 {
			msg += fmt.Sprintf("%s offline, ", english.Plural(totalOfflineNodes, "node", ""))
		}
		// Summary on total no of online and total
		// number of offline drives at the Cluster level
		msg += fmt.Sprintf("%s online, %s offline, EC:%d\n",
			english.Plural(u.Info.Backend.OnlineDisks, "drive", ""),
			english.Plural(u.Info.Backend.OfflineDisks, "drive", ""),
			u.Info.Backend.StandardSCParity)
	}

	// Remove the last new line if any
	// since this is a String() function
	msg = strings.TrimSuffix(msg, "\n")
	return
}

// JSON jsonifies service status message.
func (u clusterStruct) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(u, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate arguments passed by a user
func checkAdminInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainAdminInfo(ctx *cli.Context) error {
	checkAdminInfoSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	clusterInfo := clusterStruct{
		onlyOffline: ctx.Bool("offline"),
	}

	// Fetch info of all servers (cluster or single server)
	admInfo, e := client.ServerInfo(globalContext)
	if e != nil {
		clusterInfo.Status = "error"
		clusterInfo.Error = e.Error()
	} else {
		clusterInfo.Status = "success"
		clusterInfo.Error = ""
	}

	clusterInfo.Info = admInfo
	printMsg(clusterInfo)

	return nil
}
