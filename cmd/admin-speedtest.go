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
	msg += fmt.Sprintf("MinIO %s, %d servers, %d drives\n", s.Version, s.Servers, s.Disks)
	if globalSpeedTestVerbose {
		msg += "\n"
	}
	msg += fmt.Sprintf("PUT: %s/s, %d objs/s\n", humanize.IBytes(uint64(s.PUTStats.ThroughputPerSec)), s.PUTStats.ObjectsPerSec)
	if globalSpeedTestVerbose {
		for _, node := range s.PUTStats.Servers {
			msg += fmt.Sprintf("   * %s:, %s/s, %d objs/s", node.Endpoint, humanize.IBytes(uint64(node.ThroughputPerSec)), node.ObjectsPerSec)
			if node.Err != "" {
				msg += " error: " + node.Err
			}
			msg += "\n"

		}
	}
	if globalSpeedTestVerbose {
		msg += "\n"
	}
	msg += fmt.Sprintf("GET: %s/s, %d objs/s\n", humanize.IBytes(uint64(s.GETStats.ThroughputPerSec)), s.GETStats.ObjectsPerSec)
	if globalSpeedTestVerbose {
		for _, node := range s.GETStats.Servers {
			msg += fmt.Sprintf("   * %s:, %s/s, %d objs/s", node.Endpoint, humanize.IBytes(uint64(node.ThroughputPerSec)), node.ObjectsPerSec)
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

	autotune := false

	if ctx.NumFlags() == 0 {
		autotune = true
	}

	if ctx.NumFlags() == 2 && globalSpeedTestVerbose {
		autotune = true
	}

	result, _ := client.Speedtest(ctxt, madmin.SpeedtestOpts{
		Size:        int(size),
		Duration:    duration,
		Concurrency: concurrent,
		Autotune:    autotune,
	})

	printMsg(speedTestResult(result))

	return nil
}
