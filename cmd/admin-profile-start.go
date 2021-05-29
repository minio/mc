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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/console"
)

var adminProfileStartFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "type",
		Usage: "start profiler type, possible values are 'cpu', 'mem', 'block', 'mutex', 'trace', 'threads' and 'goroutines'",
		Value: "cpu,mem,block,goroutines",
	},
}

var adminProfileStartCmd = cli.Command{
	Name:            "start",
	Usage:           "start recording profile data",
	Action:          mainAdminProfileStart,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminProfileStartFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Start CPU profiling only
       {{.Prompt}} {{.HelpName}} --type cpu myminio/

    2. Start CPU, Memory and Block profiling concurrently
       {{.Prompt}} {{.HelpName}} --type cpu,mem,block myminio/
`,
}

func checkAdminProfileStartSyntax(ctx *cli.Context) {
	// Check flags combinations
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "start", 1) // last argument is exit code
	}

	s := set.NewStringSet()
	supportedProfilerTypes := []madmin.ProfilerType{
		madmin.ProfilerCPU,
		madmin.ProfilerMEM,
		madmin.ProfilerBlock,
		madmin.ProfilerMutex,
		madmin.ProfilerTrace,
		madmin.ProfilerThreads,
		madmin.ProfilerGoroutines,
	}
	for _, profilerType := range supportedProfilerTypes {
		s.Add(string(profilerType))
	}
	// Check if the provided profiler type is known and supported
	supportedProfiler := false
	profilers := strings.Split(strings.ToLower(ctx.String("type")), ",")
	for _, profiler := range profilers {
		if profiler != "" {
			if s.Contains(profiler) {
				supportedProfiler = true
				break
			}
		}
	}
	if !supportedProfiler {
		fatalIf(errDummy().Trace(ctx.String("type")),
			"Profiler type unrecognized. Possible values are: %v.", supportedProfilerTypes)
	}
}

// mainAdminProfileStart - the entry function of profile command
func mainAdminProfileStart(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfileStartSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	profilers := ctx.String("type")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	// Start profile
	_, cmdErr := client.StartProfiling(globalContext, madmin.ProfilerType(profilers))
	fatalIf(probe.NewError(cmdErr), "Unable to start profile.")

	console.Infoln("Profile data successfully started.")
	return nil
}
