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
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var supportPerfFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "duration",
		Usage: "duration the entire perf tests are run",
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
	cli.StringFlag{
		Name:  "filesize",
		Usage: "total amount of data read/written to each drive",
		Value: "1GiB",
	},
	cli.StringFlag{
		Name:  "blocksize",
		Usage: "read/write block size",
		Value: "4MiB",
	},
	cli.BoolFlag{
		Name:  "serial",
		Usage: "run tests on drive(s) one-by-one",
	},
}

var supportPerfCmd = cli.Command{
	Name:            "perf",
	Usage:           "analyze object, network and drive performance",
	Action:          mainSupportPerf,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportPerfFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND] [FLAGS] TARGET

COMMAND:
  drive  measure speed of drive in a cluster
  object measure speed of reading and writing object in a cluster
  net    measure network throughput of all nodes

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Run object speed measurement with autotuning the concurrency to obtain maximum throughput and IOPs:
     {{.Prompt}} {{.HelpName}} object myminio/

  2. Run object speed measurement for 20 seconds with object size of 128MiB with autotuning the concurrency to obtain maximum throughput:
     {{.Prompt}} {{.HelpName}} object myminio/ --duration 20s --size 128MiB

  3. Run drive speed measurements on all drive on all nodes (with default blockSize of 4MiB):
     {{.Prompt}} {{.HelpName}} drive myminio/

  4. Run drive speed measurements with blocksize of 64KiB, and 2GiB of data read/written from each drive:
     {{.Prompt}} {{.HelpName}} drive myminio/ --blocksize 64KiB --filesize 2GiB

  5. Run network throughput test:
     {{.Prompt}} {{.HelpName}} net myminio

`,
}

func (s speedTestResult) StringVerbose() (msg string) {
	result := s.result
	if globalPerfTestVerbose {
		msg += "\n\n"
		msg += "PUT:\n"
		for _, node := range result.PUTStats.Servers {
			msg += fmt.Sprintf("   * %s: %s/s %s objs/s\n", node.Endpoint, humanize.IBytes(node.ThroughputPerSec), humanize.Comma(int64(node.ObjectsPerSec)))
			if node.Err != "" {
				msg += " error: " + node.Err
			}
		}

		msg += "GET:\n"
		for _, node := range result.GETStats.Servers {
			msg += fmt.Sprintf("   * %s: %s/s %s objs/s\n", node.Endpoint, humanize.IBytes(node.ThroughputPerSec), humanize.Comma(int64(node.ObjectsPerSec)))
			if node.Err != "" {
				msg += " error: " + node.Err
			}
		}

	}
	return msg
}

func (s speedTestResult) String() (msg string) {
	result := s.result
	msg += fmt.Sprintf("MinIO %s, %d servers, %d drives, %s objects, %d threads",
		result.Version, result.Servers, result.Disks,
		humanize.IBytes(uint64(result.Size)), result.Concurrent)

	return msg
}

func (s speedTestResult) JSON() string {
	JSONBytes, e := json.MarshalIndent(s.result, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(JSONBytes)
}

var globalPerfTestVerbose bool

func mainSupportPerf(ctx *cli.Context) error {
	args := ctx.Args()

	// the alias parameter from cli
	aliasedURL := ""
	switch len(args) {
	case 1:
		// cannot use alias by the name 'drive' or 'net'
		if args[0] == "drive" || args[0] == "net" {
			cli.ShowCommandHelpAndExit(ctx, "perf", 1)
		}
		aliasedURL = args[0]
	case 2:
		switch args[0] {
		case "drive":
			aliasedURL = args[1]
			return mainAdminSpeedtestDrive(ctx, aliasedURL)
		case "object":
			aliasedURL = args[1]
		case "net":
			aliasedURL = args[1]
			return mainAdminSpeedtestNetperf(ctx, aliasedURL)
		default:
			cli.ShowCommandHelpAndExit(ctx, "perf", 1) // last argument is exit code
		}
	default:
		cli.ShowCommandHelpAndExit(ctx, "perf", 1) // last argument is exit code
	}

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
	globalPerfTestVerbose = ctx.Bool("verbose")

	// Turn-off autotuning only when "concurrent" is specified
	// in all other scenarios keep auto-tuning on.
	autotune := !ctx.IsSet("concurrent")

	resultCh, err := client.Speedtest(ctxt, madmin.SpeedtestOpts{
		Size:        int(size),
		Duration:    duration,
		Concurrency: concurrent,
		Autotune:    autotune,
	})
	fatalIf(probe.NewError(err), "Failed to execute performance test")

	if globalJSON {
		for result := range resultCh {
			if result.Version == "" {
				continue
			}
			printMsg(speedTestResult{
				result: result,
			})
		}
		return nil
	}

	done := make(chan struct{})

	p := tea.NewProgram(initSpeedTestUI())
	go func() {
		if e := p.Start(); e != nil {
			os.Exit(1)
		}
		close(done)
	}()

	go func() {
		var result madmin.SpeedTestResult
		for result = range resultCh {
			if result.Version == "" {
				continue
			}
			p.Send(speedTestResult{
				result: result,
			})
		}
		p.Send(speedTestResult{
			result: result,
			final:  true,
		})
	}()

	<-done

	return nil
}
