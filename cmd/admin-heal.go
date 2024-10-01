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
	"bufio"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

const (
	scanNormalMode = "normal"
	scanDeepMode   = "deep"
)

var adminHealFlags = []cli.Flag{
	cli.IntFlag{
		Name:   "pool",
		Usage:  "heal only the given pool",
		Hidden: true,
	},
	cli.IntFlag{
		Name:   "set",
		Usage:  "heal only the given set",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "scan",
		Usage:  "select the healing scan mode (normal/deep)",
		Value:  scanNormalMode,
		Hidden: true,
	},
	cli.BoolFlag{
		Name:   "recursive, r",
		Usage:  "heal recursively",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:   "dry-run, n",
		Usage:  "only inspect data, but do not mutate",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:   "force-start, f",
		Usage:  "force start a new heal sequence",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:   "force-stop, s",
		Usage:  "force stop a running heal sequence",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:  "force",
		Usage: "avoid showing a warning prompt",
	},
	cli.BoolFlag{
		Name:   "remove",
		Usage:  "remove dangling objects in heal sequence",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "storage-class",
		Usage:  "show server/drives failure tolerance with the given storage class",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:   "rewrite",
		Usage:  "rewrite objects from older to newer format",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "show verbose information",
	},
	cli.BoolFlag{
		Name:  "all-drives, a",
		Usage: "select all drives for verbose printing",
	},
}

var adminHealCmd = cli.Command{
	Name:            "heal",
	Usage:           "monitor healing for bucket(s) and object(s) on MinIO server",
	Action:          mainAdminHeal,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminHealFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Monitor healing status on a running server at alias 'myminio':
     {{.Prompt}} {{.HelpName}} myminio/
`,
}

func checkAdminHealSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	// Check for scan argument
	scanArg := ctx.String("scan")
	scanArg = strings.ToLower(scanArg)
	if scanArg != scanNormalMode && scanArg != scanDeepMode {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// stopHealMessage is container for stop heal success and failure messages.
type stopHealMessage struct {
	Status string `json:"status"`
	Alias  string `json:"alias"`
}

// String colorized stop heal message.
func (s stopHealMessage) String() string {
	return console.Colorize("HealStopped", "Heal stopped successfully at `"+s.Alias+"`.")
}

// JSON jsonified stop heal message.
func (s stopHealMessage) JSON() string {
	stopHealJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(stopHealJSONBytes)
}

type setIndex struct {
	pool, set int
}

type poolInfo struct {
	tolerance int
	endpoints []string
}

type setInfo struct {
	totalDisks int

	readyDisksCount     int    // disks online and not in healing state
	readyDisksUsedSpace uint64 // the total used space of ready disks

	incapableDisks int
}

type serverInfo struct {
	pool  int
	disks []madmin.Disk
}

func (s serverInfo) onlineDisksForSet(index setIndex) (setFound bool, count int) {
	for _, disk := range s.disks {
		if disk.PoolIndex != index.pool || disk.SetIndex != index.set {
			continue
		}
		setFound = true
		if disk.State == "ok" && !disk.Healing {
			count++
		}
	}
	return
}

// Get all drives from set statuses
func getAllDisks(sets []madmin.SetStatus) []madmin.Disk {
	var disks []madmin.Disk
	for _, set := range sets {
		disks = append(disks, set.Disks...)
	}
	return disks
}

// Get all pools id from all drives
func getPoolsIndexes(disks []madmin.Disk) []int {
	m := make(map[int]struct{})
	for _, d := range disks {
		m[d.PoolIndex] = struct{}{}
	}
	var pools []int
	for pool := range m {
		pools = append(pools, pool)
	}
	sort.Ints(pools)
	return pools
}

// Generate sets info from disks
func generateSetsStatus(disks []madmin.Disk) map[setIndex]setInfo {
	m := make(map[setIndex]setInfo)
	for _, d := range disks {
		idx := setIndex{pool: d.PoolIndex, set: d.SetIndex}
		setSt, ok := m[idx]
		if !ok {
			setSt = setInfo{}
		}
		setSt.totalDisks++
		if d.State != "ok" || d.Healing {
			setSt.incapableDisks++
		} else {
			setSt.readyDisksCount++
			setSt.readyDisksUsedSpace += d.UsedSpace
		}
		m[idx] = setSt
	}
	return m
}

// Return a map of server endpoints and the corresponding status
func generateServersStatus(disks []madmin.Disk) map[string]serverInfo {
	m := make(map[string]serverInfo)
	for _, d := range disks {
		u, e := url.Parse(d.Endpoint)
		if e != nil {
			continue
		}
		endpoint := u.Host
		if endpoint == "" {
			endpoint = "local-pool" + humanize.Ordinal(d.PoolIndex+1)
		}
		serverSt, ok := m[endpoint]
		if !ok {
			serverSt = serverInfo{
				pool: d.PoolIndex,
			}
		}
		serverSt.disks = append(serverSt.disks, d)
		m[endpoint] = serverSt
	}
	return m
}

// Return the list of endpoints of a given pool index
func computePoolEndpoints(pool int, serversStatus map[string]serverInfo) []string {
	var endpoints []string
	for endpoint, server := range serversStatus {
		if server.pool != pool {
			continue
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints
}

// Compute the tolerance of each node in a given pool
func computePoolTolerance(pool, parity int, setsStatus map[setIndex]setInfo, serversStatus map[string]serverInfo) int {
	var (
		onlineDisksPerSet = make(map[setIndex]int)
		tolerancePerSet   = make(map[setIndex]int)
	)

	for set, setStatus := range setsStatus {
		if set.pool != pool {
			continue
		}

		onlineDisksPerSet[set] = setStatus.totalDisks - setStatus.incapableDisks
		tolerancePerSet[set] = 0

		for _, server := range serversStatus {
			if server.pool != pool {
				continue
			}

			canShutdown := true
			setFound, count := server.onlineDisksForSet(set)
			if !setFound {
				continue
			}
			minDisks := setStatus.totalDisks - parity
			if onlineDisksPerSet[set]-count < minDisks {
				canShutdown = false
			}
			if canShutdown {
				tolerancePerSet[set]++
				onlineDisksPerSet[set] -= count
			} else {
				break
			}
		}
	}

	minServerTolerance := len(serversStatus)
	for _, tolerance := range tolerancePerSet {
		if tolerance < minServerTolerance {
			minServerTolerance = tolerance
		}
	}

	return minServerTolerance
}

// Extract offline nodes from offline full path endpoints
func getOfflineNodes(endpoints []string) map[string]struct{} {
	offlineNodes := make(map[string]struct{})
	for _, endpoint := range endpoints {
		offlineNodes[endpoint] = struct{}{}
	}
	return offlineNodes
}

// verboseBackgroundHealStatusMessage is container for stop heal success and failure messages.
type verboseBackgroundHealStatusMessage struct {
	Status   string `json:"status"`
	HealInfo madmin.BgHealState

	allDrives bool

	// Specify storage class to show servers/disks tolerance
	ToleranceForSC string `json:"-"`
}

// String colorized to show background heal status message.
func (s verboseBackgroundHealStatusMessage) String() string {
	var msg strings.Builder

	parity, showTolerance := s.HealInfo.SCParity[s.ToleranceForSC]
	offlineEndpoints := getOfflineNodes(s.HealInfo.OfflineEndpoints)
	allDisks := getAllDisks(s.HealInfo.Sets)
	pools := getPoolsIndexes(allDisks)
	setsStatus := generateSetsStatus(allDisks)
	serversStatus := generateServersStatus(allDisks)

	poolsInfo := make(map[int]poolInfo)
	for _, pool := range pools {
		tolerance := computePoolTolerance(pool, parity, setsStatus, serversStatus)
		endpoints := computePoolEndpoints(pool, serversStatus)
		poolsInfo[pool] = poolInfo{tolerance: tolerance, endpoints: endpoints}
	}

	distributed := len(serversStatus) > 1

	plural := ""
	if distributed {
		plural = "s"
	}
	fmt.Fprintf(&msg, "Server%s status:\n", plural)
	fmt.Fprintf(&msg, "==============\n")

	for _, pool := range pools {
		fmt.Fprintf(&msg, "Pool %s:\n", humanize.Ordinal(pool+1))

		// Sort servers in this pool by name
		orderedEndpoints := make([]string, len(poolsInfo[pool].endpoints))
		copy(orderedEndpoints, poolsInfo[pool].endpoints)
		sort.Strings(orderedEndpoints)

		for _, endpoint := range orderedEndpoints {
			// Print offline status if node is offline
			_, ok := offlineEndpoints[endpoint]
			if ok {
				stateText := console.Colorize("NodeFailed", "OFFLINE")
				fmt.Fprintf(&msg, "  %s: %s\n", endpoint, stateText)
				continue
			}
			var serverHeader strings.Builder
			var serverHeaderPrinted bool
			serverStatus := serversStatus[endpoint]
			switch {
			case showTolerance:
				hdr := "  %s: (Tolerance: %d server(s))\n"
				fmt.Fprintf(&serverHeader, hdr, endpoint, poolsInfo[serverStatus.pool].tolerance)
			default:
				hdr := "  %s:\n"
				fmt.Fprintf(&serverHeader, hdr, endpoint)
			}

			for _, d := range serverStatus.disks {
				if d.PoolIndex != pool {
					continue
				}
				stateText := ""
				switch {
				case d.State == "ok" && d.Healing:
					stateText = console.Colorize("DiskHealing", "HEALING")
				case d.State == "ok":
					if !s.allDrives {
						continue
					}
					stateText = console.Colorize("DiskOK", "OK")
				default:
					stateText = console.Colorize("DiskFailed", d.State)
				}
				if !serverHeaderPrinted {
					serverHeaderPrinted = true
					fmt.Fprint(&msg, serverHeader.String())
				}
				drivePath := d.DrivePath
				if drivePath == "" {
					if u, e := url.Parse(d.Endpoint); e == nil {
						drivePath = u.Path
					}
				}
				fmt.Fprintf(&msg, "  +  %s : %s\n", drivePath, stateText)

				thisSet := setsStatus[setIndex{d.PoolIndex, d.SetIndex}]
				if d.Healing && d.HealInfo != nil && !d.HealInfo.Finished {
					refUsedSpace := uint64(math.MaxUint64)
					if thisSet.readyDisksCount > 0 { // to avoid crashing
						refUsedSpace = thisSet.readyDisksUsedSpace / uint64(thisSet.readyDisksCount)
					}
					if refUsedSpace < d.UsedSpace { // normalize
						refUsedSpace = d.UsedSpace
					}
					fmt.Fprintf(&msg, "  |__   Progress: %d%%\n", 100*d.UsedSpace/refUsedSpace)
					fmt.Fprintf(&msg, "  |__    Started: %s\n", humanize.Time(d.HealInfo.Started))
					if d.HealInfo.RetryAttempts > 0 {
						fmt.Fprintf(&msg, "  |__    Retries: %d\n", d.HealInfo.RetryAttempts)
					}
				}
				fmt.Fprintf(&msg, "  |__   Capacity: %s/%s\n", humanize.IBytes(d.UsedSpace), humanize.IBytes(d.TotalSpace))
				if showTolerance {
					fmt.Fprintf(&msg, "  |__  Tolerance: %d drive(s)\n", parity-thisSet.incapableDisks)
				}
			}

			if serverHeaderPrinted {
				fmt.Fprintf(&msg, "\n")
			}
		}
	}

	if showTolerance {
		fmt.Fprintf(&msg, "\n")
		fmt.Fprintf(&msg, "Server Failure Tolerance:\n")
		fmt.Fprintf(&msg, "========================\n")
		for i, pool := range poolsInfo {
			fmt.Fprintf(&msg, "Pool %s:\n", humanize.Ordinal(i+1))
			fmt.Fprintf(&msg, "   Tolerance : %d server(s)\n", pool.tolerance)
			fmt.Fprintf(&msg, "       Nodes :")
			for _, endpoint := range pool.endpoints {
				fmt.Fprintf(&msg, " %s", endpoint)
			}
			fmt.Fprintf(&msg, "\n")
		}
	}

	summary := shortBackgroundHealStatusMessage{HealInfo: s.HealInfo}

	fmt.Fprint(&msg, "\n")
	fmt.Fprint(&msg, "Summary:\n")
	fmt.Fprint(&msg, "=======\n")
	fmt.Fprint(&msg, summary.String())
	fmt.Fprint(&msg, "\n")

	return msg.String()
}

// JSON jsonified stop heal message.
func (s verboseBackgroundHealStatusMessage) JSON() string {
	healJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(healJSONBytes)
}

// shortBackgroundHealStatusMessage is container for stop heal success and failure messages.
type shortBackgroundHealStatusMessage struct {
	Status   string `json:"status"`
	HealInfo madmin.BgHealState
}

// String colorized to show background heal status message.
func (s shortBackgroundHealStatusMessage) String() string {
	healPrettyMsg := ""
	var (
		itemsHealed        uint64
		bytesHealed        uint64
		itemsFailed        uint64
		bytesFailed        uint64
		itemsHealedPerSec  float64
		bytesHealedPerSec  float64
		startedAt          time.Time
		setsExceedsStd     int
		setsExceedsReduced int

		// The addition of Elapsed time of each parallel healing operation
		// this is needed to calculate the rate of healing
		accumulatedElapsedTime time.Duration
	)

	var problematicDisks int
	leastPct := 100.0

	for _, set := range s.HealInfo.Sets {
		setsStatus := generateSetsStatus(set.Disks)
		// Furthest along disk...
		var furthestHealingDisk *madmin.Disk
		missingInSet := 0
		for _, disk := range set.Disks {
			// Ignore disk with non 'ok' status
			if disk.State != madmin.DriveStateOk {
				if disk.State != madmin.DriveStateUnformatted {
					missingInSet++
					problematicDisks++
				}
				continue
			}

			if disk.HealInfo != nil && !disk.HealInfo.Finished {
				missingInSet++

				thisSet := setsStatus[setIndex{pool: disk.PoolIndex, set: disk.SetIndex}]
				refUsedSpace := uint64(math.MaxUint64)
				if thisSet.readyDisksCount > 0 {
					refUsedSpace = thisSet.readyDisksUsedSpace / uint64(thisSet.readyDisksCount)
				}
				if refUsedSpace < disk.UsedSpace {
					refUsedSpace = disk.UsedSpace
				}
				if refUsedSpace > 0 {
					if pct := float64(disk.UsedSpace) / float64(refUsedSpace); pct < leastPct {
						leastPct = pct
					}
				} else {
					// Unlikely to have max used space in an erasure set to be zero, but still set this to zero
					leastPct = 0
				}

				disk := disk
				if furthestHealingDisk == nil {
					furthestHealingDisk = &disk
					continue
				}
				if disk.HealInfo.ItemsHealed+disk.HealInfo.ItemsFailed > furthestHealingDisk.HealInfo.ItemsHealed+furthestHealingDisk.HealInfo.ItemsFailed {
					furthestHealingDisk = &disk
					continue
				}
			}
		}

		if furthestHealingDisk != nil {
			disk := furthestHealingDisk

			// Approximate values
			itemsHealed += disk.HealInfo.ItemsHealed
			bytesHealed += disk.HealInfo.BytesDone
			bytesFailed += disk.HealInfo.BytesFailed
			itemsFailed += disk.HealInfo.ItemsFailed

			if !disk.HealInfo.Started.IsZero() {
				if !disk.HealInfo.Started.Before(startedAt) {
					startedAt = disk.HealInfo.Started
				}

				if !disk.HealInfo.LastUpdate.IsZero() {
					accumulatedElapsedTime += disk.HealInfo.LastUpdate.Sub(disk.HealInfo.Started)
				}

				bytesHealedPerSec += float64(time.Second) * float64(disk.HealInfo.BytesDone) / float64(disk.HealInfo.LastUpdate.Sub(disk.HealInfo.Started))
				itemsHealedPerSec += float64(time.Second) * float64(disk.HealInfo.ItemsHealed+disk.HealInfo.ItemsFailed) / float64(disk.HealInfo.LastUpdate.Sub(disk.HealInfo.Started))

			}
			if n, ok := s.HealInfo.SCParity["STANDARD"]; ok && missingInSet > n {
				setsExceedsStd++
			}
			if n, ok := s.HealInfo.SCParity["REDUCED_REDUNDANCY"]; ok && missingInSet > n {
				setsExceedsReduced++
			}
		}
	}

	if startedAt.IsZero() && itemsHealed == 0 {
		healPrettyMsg += "No active healing is detected for new disks"
		if problematicDisks > 0 {
			healPrettyMsg += fmt.Sprintf(", though %d offline disk(s) found.", problematicDisks)
		} else {
			healPrettyMsg += "."
		}
		return healPrettyMsg
	}

	// Objects healed information
	healPrettyMsg += fmt.Sprintf("Objects Healed: %s, %s (%s)\n",
		humanize.Comma(int64(itemsHealed)), humanize.IBytes(bytesHealed), humanize.CommafWithDigits(leastPct*100, 1)+"%")
	healPrettyMsg += fmt.Sprintf("Objects Failed: %s\n", humanize.Comma(int64(itemsFailed)))

	if accumulatedElapsedTime > 0 {
		healPrettyMsg += fmt.Sprintf("Heal rate: %d obj/s, %s/s\n", int64(itemsHealedPerSec), humanize.IBytes(uint64(bytesHealedPerSec)))
	}

	if problematicDisks > 0 {
		healPrettyMsg += "\n"
		healPrettyMsg += fmt.Sprintf("%d offline disk(s) found.", problematicDisks)
	}
	if setsExceedsStd > 0 {
		healPrettyMsg += "\n"
		healPrettyMsg += fmt.Sprintf("%d of %d sets exceeds standard parity count EC:%d lost/offline disks", setsExceedsStd, len(s.HealInfo.Sets), s.HealInfo.SCParity["STANDARD"])
	}
	if setsExceedsReduced > 0 {
		healPrettyMsg += "\n"
		healPrettyMsg += fmt.Sprintf("%d of %d sets exceeds reduced parity count EC:%d lost/offline disks", setsExceedsReduced, len(s.HealInfo.Sets), s.HealInfo.SCParity["REDUCED_REDUNDANCY"])
	}
	return healPrettyMsg
}

// JSON jsonified stop heal message.
func (s shortBackgroundHealStatusMessage) JSON() string {
	healJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(healJSONBytes)
}

func transformScanArg(scanArg string) madmin.HealScanMode {
	switch scanArg {
	case "deep":
		return madmin.HealDeepScan
	}
	return madmin.HealNormalScan
}

// mainAdminHeal - the entry function of heal command
func mainAdminHeal(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminHealSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	console.SetColor("Heal", color.New(color.FgGreen, color.Bold))
	console.SetColor("Dot", color.New(color.FgGreen, color.Bold))
	console.SetColor("HealBackgroundTitle", color.New(color.FgGreen, color.Bold))
	console.SetColor("HealBackground", color.New(color.Bold))
	console.SetColor("HealUpdateUI", color.New(color.FgYellow, color.Bold))
	console.SetColor("HealStopped", color.New(color.FgGreen, color.Bold))

	console.SetColor("DiskHealing", color.New(color.FgYellow, color.Bold))
	console.SetColor("DiskOK", color.New(color.FgGreen, color.Bold))
	console.SetColor("DiskFailed", color.New(color.FgRed, color.Bold))
	console.SetColor("NodeFailed", color.New(color.FgRed, color.Bold))

	// Create a new MinIO Admin Client
	adminClnt, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, prefix := splits[1], splits[2]

	clnt, err := newClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(clnt.GetURL().String()), "Unable to create client for URL ", aliasedURL)
		return nil
	}

	// Return the background heal status when the user
	// doesn't pass a bucket or --recursive flag.
	if bucket == "" && !ctx.Bool("recursive") {
		bgHealStatus, e := adminClnt.BackgroundHealStatus(globalContext)
		fatalIf(probe.NewError(e), "Unable to get background heal status.")
		if ctx.Bool("verbose") {
			printMsg(verboseBackgroundHealStatusMessage{
				Status:         "success",
				HealInfo:       bgHealStatus,
				allDrives:      ctx.Bool("all-drives"),
				ToleranceForSC: strings.ToUpper(ctx.String("storage-class")),
			})
		} else {
			printMsg(shortBackgroundHealStatusMessage{
				Status:   "success",
				HealInfo: bgHealStatus,
			})
		}
		return nil
	}

	opts := madmin.HealOpts{
		ScanMode:  transformScanArg(ctx.String("scan")),
		Remove:    ctx.Bool("remove"),
		Recursive: ctx.Bool("recursive"),
		DryRun:    ctx.Bool("dry-run"),
		Recreate:  ctx.Bool("rewrite"),
	}

	if ctx.IsSet("pool") {
		p := ctx.Int("pool")
		if p < 1 {
			fatalIf(errInvalidArgument(), "--pool takes a non zero positive number.")
		}
		p--
		opts.Pool = &p
	}

	if ctx.IsSet("set") {
		s := ctx.Int("set")
		if s < 1 {
			fatalIf(errInvalidArgument(), "--set takes a non zero positive number.")
		}
		s--
		opts.Set = &s
	}

	forceStart := ctx.Bool("force-start")
	forceStop := ctx.Bool("force-stop")
	if forceStop {
		_, _, e := adminClnt.Heal(globalContext, bucket, prefix, opts, "", forceStart, forceStop)
		fatalIf(probe.NewError(e), "Unable to stop healing.")
		printMsg(stopHealMessage{Status: "success", Alias: aliasedURL})
		return nil
	}

	if opts.Recursive && opts.Pool == nil && opts.Set == nil && isTerminal() && !ctx.Bool("force") {
		fmt.Printf("You are about to scan and heal the whole namespace in all pools and sets, please confirm [y/N]: ")
		answer, e := bufio.NewReader(os.Stdin).ReadString('\n')
		fatalIf(probe.NewError(e), "Unable to parse user input.")
		if answer = strings.TrimSpace(strings.ToLower(answer)); answer != "y" && answer != "yes" {
			fmt.Println("Heal aborted!")
			return nil
		}
	}

	healStart, _, e := adminClnt.Heal(globalContext, bucket, prefix, opts, "", forceStart, false)
	fatalIf(probe.NewError(e), "Unable to start healing.")

	ui := uiData{
		Bucket:                bucket,
		Prefix:                prefix,
		Client:                adminClnt,
		ClientToken:           healStart.ClientToken,
		ForceStart:            forceStart,
		HealOpts:              &opts,
		ObjectsByOnlineDrives: make(map[int]int64),
		HealthCols:            make(map[col]int64),
		CurChan:               cursorAnimate(),
	}

	res, e := ui.DisplayAndFollowHealStatus(aliasedURL)
	if e != nil {
		if res.FailureDetail != "" {
			data, _ := json.MarshalIndent(res, "", " ")
			traceStr := string(data)
			fatalIf(probe.NewError(e).Trace(aliasedURL, traceStr), "Unable to display heal status.")
		} else {
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to display heal status.")
		}
	}
	return nil
}
