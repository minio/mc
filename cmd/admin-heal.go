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
	"path/filepath"
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
     Objects Healed: 7/27 (25.9%), 31 MB/110 MB (0%)
     Heal rate: 3 obj/s, 10 MB/s
     Estimated Completion: 11 seconds
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

// backgroundHealStatusMessage is container for stop heal success and failure messages.
type backgroundHealStatusMessage struct {
	Status   string `json:"status"`
	HealInfo madmin.BgHealState
}

// String colorized to show background heal status message.
func (s backgroundHealStatusMessage) String() string {
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

	if startedAt.IsZero() {
		healPrettyMsg += "No ongoing active healing."
		return healPrettyMsg
	}

	if totalItems > 0 && totalBytes > 0 {
		// Objects healed information
		itemsPct := 100 * float64(itemsHealed) / float64(totalItems)
		bytesPct := 100 * float64(bytesHealed) / float64(totalBytes)

		healPrettyMsg += fmt.Sprintf("Objects Healed: %s/%s (%s%%), %s/%s (%s%%)\n",
			humanize.Comma(int64(itemsHealed)), humanize.Comma(int64(totalItems)), humanize.CommafWithDigits(itemsPct, 1),
			humanize.Bytes(bytesHealed), humanize.Bytes(totalBytes), humanize.CommafWithDigits(bytesPct, 1))
	} else {
		healPrettyMsg += fmt.Sprintf("Objects Healed: %s, %s\n", humanize.Comma(int64(itemsHealed)), humanize.Bytes(bytesHealed))
	}

	bytesHealedPerSec := float64(uint64(time.Second)*bytesHealed) / float64(accumulatedElapsedTime)
	itemsHealedPerSec := float64(uint64(time.Second)*itemsHealed) / float64(accumulatedElapsedTime)
	healPrettyMsg += fmt.Sprintf("Heal rate: %d obj/s, %s/s\n", int64(itemsHealedPerSec), humanize.IBytes(uint64(bytesHealedPerSec)))

	if totalItems > 0 && totalBytes > 0 {
		// Estimation completion
		avgTimePerObject := float64(accumulatedElapsedTime) / float64(itemsHealed)
		estimatedDuration := time.Duration(avgTimePerObject * float64(totalItems))
		estimatedFinishTime := startedAt.Add(estimatedDuration)
		healPrettyMsg += fmt.Sprintf("Estimated Completion: %s\n", humanize.RelTime(now, estimatedFinishTime, "", ""))
	}

	return healPrettyMsg
}

// JSON jsonified stop heal message.
func (s backgroundHealStatusMessage) JSON() string {
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
		printMsg(backgroundHealStatusMessage{Status: "success", HealInfo: bgHealStatus})
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
