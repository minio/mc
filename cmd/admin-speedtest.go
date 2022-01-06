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
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminSpeedtestFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "duration",
		Usage: "duration the entire speedtests are run",
		Value: "10s",
	},
	cli.StringFlag{
		Name:  "size",
		Usage: "size of the object used for uploads/downloads",
		Value: "64MiB",
	},
	cli.IntFlag{
		Name:  "concurrent",
		Usage: "number of concurrent requests per server",
		Value: 32,
	},
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "Show per-server stats",
	},
}

var adminSpeedtestCmd = cli.Command{
	Name:            "speedtest",
	Usage:           "Run server side speed test",
	Action:          mainAdminSpeedtest,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminSpeedtestFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Run speedtest with autotuning the concurrency to figure out the maximum throughput and iops values:
     {{.Prompt}} {{.HelpName}} myminio/

  2. Run speedtest for 20 seconds with object size of 128MiB, 32 concurrent requests per server:
     {{.Prompt}} {{.HelpName}} myminio/ --duration 20s --size 128MiB --concurrent 32
`,
}

type speedTestResult madmin.SpeedTestResult

func (s speedTestResult) String() (msg string) {
	msg += fmt.Sprintf("\nMinIO %s, %d servers, %d drives\n", s.Version, s.Servers, s.Disks)

	var errorReturned bool
	for _, node := range s.PUTStats.Servers {
		if node.Err != "" {
			errorReturned = true
			break
		}
	}
	for _, node := range s.GETStats.Servers {
		if node.Err != "" {
			errorReturned = true
			break
		}
	}

	// When no error is found and yet without results, this means the speedtest duration is too short
	if !errorReturned && (s.PUTStats.ThroughputPerSec == 0 || s.GETStats.ThroughputPerSec == 0) {
		msg += "\n"
		msg += "No results found for this speedtest iteration. Try increasing --duration flag."
		msg += "\n"
		return
	}

	msg += fmt.Sprintf("PUT: %s/s, %s objs/s\n", humanize.IBytes(s.PUTStats.ThroughputPerSec), humanize.Comma(int64(s.PUTStats.ObjectsPerSec)))
	if globalSpeedTestVerbose {
		for _, node := range s.PUTStats.Servers {
			msg += fmt.Sprintf("   * %s: %s/s %s objs/s", node.Endpoint, humanize.IBytes(node.ThroughputPerSec), humanize.Comma(int64(node.ObjectsPerSec)))
			if node.Err != "" {
				msg += " error: " + node.Err
			}
			msg += "\n"

		}
	}
	if globalSpeedTestVerbose {
		msg += "\n"
	}
	msg += fmt.Sprintf("GET: %s/s, %s objs/s\n", humanize.IBytes(s.GETStats.ThroughputPerSec), humanize.Comma(int64(s.GETStats.ObjectsPerSec)))
	if globalSpeedTestVerbose {
		for _, node := range s.GETStats.Servers {
			msg += fmt.Sprintf("   * %s: %s/s %s objs/s", node.Endpoint, humanize.IBytes(node.ThroughputPerSec), humanize.Comma(int64(node.ObjectsPerSec)))
			if node.Err != "" {
				msg += " error: " + node.Err
			}
			msg += "\n"
		}
	}
	return msg
}

func (s speedTestResult) JSON() string {
	JSONBytes, e := json.MarshalIndent(s, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(JSONBytes)
}

var globalSpeedTestVerbose bool

func mainAdminSpeedtest(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "speedtest", 1) // last argument is exit code
	}

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	duration, e := time.ParseDuration(ctx.String("duration"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse duration")
		return nil
	}
	if duration <= 0 {
		fatalIf(errInvalidArgument(), "duration cannot be 0 or negative")
		return nil
	}
	size, e := humanize.ParseBytes(ctx.String("size"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse object size")
		return nil
	}
	if size < 0 {
		fatalIf(errInvalidArgument(), "size is expected to be atleast 0 bytes")
		return nil
	}
	concurrent := ctx.Int("concurrent")
	if concurrent <= 0 {
		fatalIf(errInvalidArgument(), "concurrency cannot be '0' or negative")
		return nil
	}
	globalSpeedTestVerbose = ctx.Bool("verbose")

	// Turn-off autotuning only when "concurrent" is specified
	// in all other scenarios keep auto-tuning on.
	autotune := !ctx.IsSet("concurrent")

	resultCh, err := client.Speedtest(ctxt, madmin.SpeedtestOpts{
		Size:        int(size),
		Duration:    duration,
		Concurrency: concurrent,
		Autotune:    autotune,
	})
	fatalIf(probe.NewError(err), "Failed to execute speedtest")

	spinnerCh, s := startSpinner()

	var result madmin.SpeedTestResult
	for result = range resultCh {
		select {
		case spinnerCh <- struct{}{}:
		default:
		}
		if result.Version == "" {
			continue
		}
		if !globalJSON {
			s.Stop()
			close(spinnerCh)
			fmt.Printf("(With %s object size, %d concurrency) PUT: %s/s GET: %s/s\n", humanize.IBytes(uint64(result.Size)), result.Concurrent, humanize.IBytes(result.PUTStats.ThroughputPerSec), humanize.IBytes(result.GETStats.ThroughputPerSec))
			spinnerCh, s = startSpinner()
		}
	}
	if result.Version != "" {
		s.Stop()
		close(spinnerCh)
		printMsg(speedTestResult(result))
	}

	return nil
}

func startSpinner() (chan struct{}, *spinner.Spinner) {
	ch := make(chan struct{}, 1)
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	firstTimeDelay := true
	go func() {
		for {
			if !globalJSON {
				s.Suffix = " Running speedtest"
				if firstTimeDelay {
					// First time delay is a work around for the case where after the last server response
					// we don't endup printing a redundant "Running speedtest" line. Because of the sleep here
					// the program would have printed results and exited.
					time.Sleep(100 * time.Millisecond)
					firstTimeDelay = false
				}
				_, ok := <-ch
				if !ok {
					return
				}
				s.Start()
				time.Sleep(500 * time.Millisecond)
				s.Stop()
			}
		}
	}()
	return ch, s
}
