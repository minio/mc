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
		Usage: "[DEPRECATED] select the healing scan mode (normal/deep)",
		Value: scanNormalMode,
	},
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "[DEPRECATED] heal recursively",
	},
	cli.BoolFlag{
		Name:  "dry-run, n",
		Usage: "[DEPRECATED] only inspect data, but do not mutate",
	},
	cli.BoolFlag{
		Name:  "force-start, f",
		Usage: "[DEPRECATED] force start a new heal sequence",
	},
	cli.BoolFlag{
		Name:  "force-stop, s",
		Usage: "[DEPRECATED] force stop a running heal sequence",
	},
	cli.BoolFlag{
		Name:  "remove",
		Usage: "[DEPRECATED] remove dangling objects in heal sequence",
	},
}

var adminHealCmd = cli.Command{
	Name:            "heal",
	Usage:           "[DEPRECATED] heal disks, buckets and objects on MinIO server",
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
SCAN MODES:
  normal (default): Heal objects which are missing on one or more disks.
  deep            : Heal objects which are missing or with silent data corruption on one or more disks.

DEPRECATED:
  MinIO server now supports auto-heal, this command will be removed in future.
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
	dot := console.Colorize("Dot", " ●  ")

	healPrettyMsg := console.Colorize("HealBackgroundTitle", "Background healing status:\n")
	healPrettyMsg += dot + fmt.Sprintf("%s item(s) scanned in total\n",
		console.Colorize("HealBackground", s.HealInfo.ScannedItemsCount))

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
	client, err := newAdminClient(aliasedURL)
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

	for content := range clnt.List(globalContext, ListOptions{Recursive: false, ShowDir: DirNone}) {
		if content.Err != nil {
			fatalIf(content.Err.Trace(clnt.GetURL().String()), "Unable to heal bucket `"+bucket+"`.")
			return nil
		}
	}

	// Return the background heal status when the user
	// doesn't pass a bucket or --recursive flag.
	if bucket == "" && !ctx.Bool("recursive") {
		bgHealStatus, berr := client.BackgroundHealStatus(globalContext)
		fatalIf(probe.NewError(berr), "Failed to get the status of the background heal.")
		printMsg(backgroundHealStatusMessage{Status: "success", HealInfo: bgHealStatus})
		return nil
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
		_, _, herr := client.Heal(globalContext, bucket, prefix, opts, "", forceStart, forceStop)
		fatalIf(probe.NewError(herr), "Failed to stop heal sequence.")
		printMsg(stopHealMessage{Status: "success", Alias: aliasedURL})
		return nil
	}

	healStart, _, herr := client.Heal(globalContext, bucket, prefix, opts, "", forceStart, false)
	fatalIf(probe.NewError(herr), "Failed to start heal sequence.")

	ui := uiData{
		Bucket:                bucket,
		Prefix:                prefix,
		Client:                client,
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
