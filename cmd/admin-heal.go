// Copyright (c) 2015-2021 MinIO, Inc.
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
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

const (
	scanNormalMode = "normal"
	scanDeepMode   = "deep"
)

var adminHealFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "scan",
		Usage: "select the healing scan mode (normal/deep)",
		Value: scanNormalMode,
	},
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "heal recursively",
	},
	cli.BoolFlag{
		Name:  "dry-run, n",
		Usage: "only inspect data, but do not mutate",
	},
	cli.BoolFlag{
		Name:  "force-start, f",
		Usage: "force start a new heal sequence",
	},
	cli.BoolFlag{
		Name:  "force-stop, s",
		Usage: "force stop a running heal sequence",
	},
	cli.BoolFlag{
		Name:  "remove",
		Usage: "remove dangling objects in heal sequence",
	},
	cli.StringFlag{
		Name:  "storage-class",
		Usage: "show server/disks failure tolerance with the given storage class",
	},
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "show verbose information",
	},
}

var adminHealCmd = cli.Command{
	Name:            "heal",
	Usage:           "heal disks, buckets and objects on MinIO server",
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
		cli.ShowCommandHelpAndExit(ctx, "heal", 1) // last argument is exit code
	}

	// Check for scan argument
	scanArg := ctx.String("scan")
	scanArg = strings.ToLower(scanArg)
	if scanArg != scanNormalMode && scanArg != scanDeepMode {
		cli.ShowCommandHelpAndExit(ctx, "heal", 1) // last argument is exit code
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

type healingStatus struct {
	started      time.Time
	totalObjects uint64
	totalHealed  uint64
}

// Estimation of when the healing will finish
func (h healingStatus) ETA() time.Time {
	if !h.started.IsZero() && h.totalObjects > h.totalHealed {
		objScanSpeed := float64(time.Now().UTC().Sub(h.started)) / float64(h.totalHealed)
		remainingDuration := float64(h.totalObjects-h.totalHealed) * objScanSpeed
		return time.Now().UTC().Add(time.Duration(remainingDuration))
	}
	return time.Time{}
}

type poolInfo struct {
	tolerance int
	endpoints []string
}

type diskInfo struct {
	set     setIndex
	path    string
	state   string
	healing bool

	usedSpace, totalSpace uint64
}

type setInfo struct {
	healingStatus  healingStatus
	totalDisks     int
	incapableDisks int
}

type serverInfo struct {
	pool  int
	disks []diskInfo
}

func (s serverInfo) onlineDisksForSet(index setIndex) (setFound bool, count int) {
	for _, disk := range s.disks {
		if disk.set != index {
			continue
		}
		setFound = true
		if disk.state == "ok" && !disk.healing {
			count++
		}
	}
	return
}

// Get all disks from set statuses
func getAllDisks(sets []madmin.SetStatus) []madmin.Disk {
	var disks []madmin.Disk
	for _, set := range sets {
		disks = append(disks, set.Disks...)
	}
	return disks
}

// Get all pools id from all disks
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
		}
		if d.Healing && d.HealInfo != nil {
			setSt.healingStatus.started = d.HealInfo.Started
			setSt.healingStatus.totalObjects = d.HealInfo.ObjectsTotalCount
			setSt.healingStatus.totalHealed = d.HealInfo.ObjectsHealed
		}
		m[idx] = setSt
	}
	return m
}

// Return a map of server endpoints and the corresponding status
func generateServersStatus(disks []madmin.Disk) map[string]serverInfo {
	m := make(map[string]serverInfo)
	for _, d := range disks {
		u, err := url.Parse(d.Endpoint)
		if err != nil {
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
		setIndex := setIndex{pool: d.PoolIndex, set: d.SetIndex}
		serverSt.disks = append(serverSt.disks, diskInfo{
			set:        setIndex,
			path:       u.Path,
			state:      d.State,
			healing:    d.Healing,
			usedSpace:  d.UsedSpace,
			totalSpace: d.TotalSpace,
		})
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
				fmt.Fprintf(&msg, fmt.Sprintf("  %s: %s\n", endpoint, stateText))
				continue
			}
			serverStatus := serversStatus[endpoint]
			switch {
			case showTolerance:
				serverHeader := "  %s: (Tolerance: %d server(s))\n"
				fmt.Fprintf(&msg, fmt.Sprintf(serverHeader, endpoint, poolsInfo[serverStatus.pool].tolerance))
			default:
				serverHeader := "  %s:\n"
				fmt.Fprintf(&msg, fmt.Sprintf(serverHeader, endpoint))
			}

			for _, d := range serverStatus.disks {
				if d.set.pool != pool {
					continue
				}
				stateText := ""
				switch {
				case d.state == "ok" && d.healing:
					stateText = console.Colorize("DiskHealing", "HEALING")
				case d.state == "ok":
					stateText = console.Colorize("DiskOK", "OK")
				default:
					stateText = console.Colorize("DiskFailed", d.state)
				}
				fmt.Fprintf(&msg, "  +  %s : %s\n", d.path, stateText)
				if d.healing {
					estimationText := "Calculating..."
					if eta := setsStatus[d.set].healingStatus.ETA(); !eta.IsZero() {
						estimationText = humanize.RelTime(time.Now().UTC(), eta, "", "")
					}
					fmt.Fprintf(&msg, "  |__ Estimated: %s\n", estimationText)
				}
				fmt.Fprintf(&msg, "  |__  Capacity: %s/%s\n", humanize.IBytes(d.usedSpace), humanize.IBytes(d.totalSpace))
				if showTolerance {
					fmt.Fprintf(&msg, "  |__ Tolerance: %d disk(s)\n", parity-setsStatus[d.set].incapableDisks)
				}
			}

			fmt.Fprintf(&msg, "\n")
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

	fmt.Fprintf(&msg, "\n")
	fmt.Fprintf(&msg, "Summary:\n")
	fmt.Fprintf(&msg, "=======\n")
	fmt.Fprintf(&msg, summary.String())
	fmt.Fprintf(&msg, "\n")

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
		totalItems  uint64
		totalBytes  uint64
		itemsHealed uint64
		bytesHealed uint64
		startedAt   time.Time

		// The addition of Elapsed time of each parallel healing operation
		accumulatedElapsedTime time.Duration
	)

	type setInfo struct {
		pool, set int
	}

	dedup := make(map[setInfo]struct{})

	for _, set := range s.HealInfo.Sets {
		for _, disk := range set.Disks {
			if disk.HealInfo != nil {
				// Avoid counting two disks beloning to the same pool/set
				diskLocation := setInfo{pool: disk.PoolIndex, set: disk.SetIndex}
				_, found := dedup[diskLocation]
				if found {
					continue
				}
				dedup[diskLocation] = struct{}{}

				// Approximate values
				totalItems += disk.HealInfo.ObjectsTotalCount
				totalBytes += disk.HealInfo.ObjectsTotalSize
				itemsHealed += disk.HealInfo.ItemsHealed
				bytesHealed += disk.HealInfo.BytesDone

				if !disk.HealInfo.Started.IsZero() && !disk.HealInfo.Started.Before(startedAt) {
					startedAt = disk.HealInfo.Started
				}

				if !disk.HealInfo.Started.IsZero() && !disk.HealInfo.LastUpdate.IsZero() {
					accumulatedElapsedTime += disk.HealInfo.LastUpdate.Sub(disk.HealInfo.Started)
				}
			}
		}
	}

	now := time.Now()

	for _, mrf := range s.HealInfo.MRF {
		totalItems += mrf.TotalItems
		totalBytes += mrf.TotalBytes
		bytesHealed += mrf.BytesHealed
		itemsHealed += mrf.ItemsHealed

		if !mrf.Started.IsZero() {
			if startedAt.IsZero() || mrf.Started.Before(startedAt) {
				startedAt = mrf.Started
			}

			accumulatedElapsedTime += now.Sub(mrf.Started)
		}
	}

	if startedAt.IsZero() && itemsHealed == 0 {
		healPrettyMsg += "No active healing in progress."
		return healPrettyMsg
	}

	if totalItems > 0 && totalBytes > 0 {
		// Objects healed information
		itemsPct := 100 * float64(itemsHealed) / float64(totalItems)
		bytesPct := 100 * float64(bytesHealed) / float64(totalBytes)

		healPrettyMsg += fmt.Sprintf("Objects Healed: %s/%s (%s), %s/%s (%s)\n",
			humanize.Comma(int64(itemsHealed)), humanize.Comma(int64(totalItems)), humanize.CommafWithDigits(itemsPct, 1)+"%%",
			humanize.Bytes(bytesHealed), humanize.Bytes(totalBytes), humanize.CommafWithDigits(bytesPct, 1)+"%%")
	} else {
		healPrettyMsg += fmt.Sprintf("Objects Healed: %s, %s\n", humanize.Comma(int64(itemsHealed)), humanize.Bytes(bytesHealed))
	}

	if accumulatedElapsedTime > 0 {
		bytesHealedPerSec := float64(uint64(time.Second)*bytesHealed) / float64(accumulatedElapsedTime)
		itemsHealedPerSec := float64(uint64(time.Second)*itemsHealed) / float64(accumulatedElapsedTime)
		healPrettyMsg += fmt.Sprintf("Heal rate: %d obj/s, %s/s\n", int64(itemsHealedPerSec), humanize.IBytes(uint64(bytesHealedPerSec)))
	}

	if totalItems > 0 && totalBytes > 0 && !startedAt.IsZero() {
		// Estimation completion
		avgTimePerObject := float64(accumulatedElapsedTime) / float64(itemsHealed)
		estimatedDuration := time.Duration(avgTimePerObject * float64(totalItems))
		estimatedFinishTime := startedAt.Add(estimatedDuration)
		healPrettyMsg += fmt.Sprintf("Estimated Completion: %s\n", humanize.RelTime(now, estimatedFinishTime, "", ""))
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
		bgHealStatus, berr := adminClnt.BackgroundHealStatus(globalContext)
		fatalIf(probe.NewError(berr), "Failed to get the status of the background heal.")
		if ctx.Bool("verbose") {
			printMsg(verboseBackgroundHealStatusMessage{
				Status:         "success",
				HealInfo:       bgHealStatus,
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

	for content := range clnt.List(globalContext, ListOptions{Recursive: false, ShowDir: DirNone}) {
		if content.Err != nil {
			fatalIf(content.Err.Trace(clnt.GetURL().String()), "Unable to heal bucket `"+bucket+"`.")
			return nil
		}
	}

	opts := madmin.HealOpts{
		ScanMode:  transformScanArg(ctx.String("scan")),
		Remove:    ctx.Bool("remove"),
		Recursive: ctx.Bool("recursive"),
		DryRun:    ctx.Bool("dry-run"),
	}

	forceStart := ctx.Bool("force-start")
	forceStop := ctx.Bool("force-stop")
	if forceStop {
		_, _, herr := adminClnt.Heal(globalContext, bucket, prefix, opts, "", forceStart, forceStop)
		fatalIf(probe.NewError(herr), "Failed to stop heal sequence.")
		printMsg(stopHealMessage{Status: "success", Alias: aliasedURL})
		return nil
	}

	healStart, _, herr := adminClnt.Heal(globalContext, bucket, prefix, opts, "", forceStart, false)
	fatalIf(probe.NewError(herr), "Failed to start heal sequence.")

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
