/*
 * MinIO Client (C) 2018 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminProfileStartFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "type",
		Usage: "start profiler type, possible values are 'cpu', 'mem', 'block', 'mutex', 'trace', 'threads' and 'goroutines'",
		Value: "cpu,mem,block",
	},
}

var adminProfileStartCmd = cli.Command{
	Name:            "start",
	Usage:           "start recording profile data",
	Action:          mainAdminProfileStart,
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
