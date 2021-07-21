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
	"encoding/json"
	"fmt"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
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
  1. Run speedtest with default values, check '--help' for default values:
     {{.Prompt}} {{.HelpName}} myminio/

  2. Run speedtest for 20seconds with object size of 128MiB, 32 concurrent requests per server:
     {{.Prompt}} {{.HelpName}} --duration 20s --size 128MiB --concurrent 32 myminio/
`,
}

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
	results, e := client.Speedtest(ctxt, madmin.SpeedtestOpts{
		Size:        int(size),
		Duration:    duration,
		Concurrency: concurrent,
	})
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to run speedtest")
		return nil
	}

	if globalJSON {
		buf, e := json.Marshal(results)
		fatalIf(probe.NewError(e), "Unable to marshal into JSON results")
		fmt.Println(string(buf))
		return nil
	}

	uploads := uint64(0)
	downloads := uint64(0)
	for _, result := range results {
		uploads += result.Uploads
		downloads += result.Downloads
	}

	durationSecs := duration.Seconds()
	uploadSpeed := humanize.IBytes(uploads * uint64(size) / uint64(durationSecs))
	downloadSpeed := humanize.IBytes(downloads * uint64(size) / uint64(durationSecs))

	fmt.Printf("Operation: PUT\n* Average: %s/s, %d objs/s\n\n",
		uploadSpeed, uploads/uint64(durationSecs))

	fmt.Printf("Throughput by host:\n")
	for _, result := range results {
		fmt.Printf(" * %s: Avg: %s/s, %d objs/s\n",
			result.Endpoint,
			humanize.IBytes(result.Uploads*uint64(size)/uint64(durationSecs)),
			result.Uploads/uint64(durationSecs),
		)
	}

	fmt.Printf("\nOperation: GET\n* Average: %s/s, %d objs/s\n\n",
		downloadSpeed, downloads/uint64(durationSecs))

	fmt.Printf("Throughput by host:\n")
	for _, result := range results {
		fmt.Printf(" * %s: Avg: %s/s, %d objs/s\n",
			result.Endpoint,
			humanize.IBytes(result.Downloads*uint64(size)/uint64(durationSecs)),
			result.Downloads/uint64(durationSecs),
		)
	}
	fmt.Println()

	return nil
}
