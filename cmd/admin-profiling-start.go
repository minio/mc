/*
 * Minio Client (C) 2018 Minio, Inc.
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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminProfilingStartFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "type",
		Usage: "Profiler type, possible values are 'cpu', 'mem', 'block', 'mutex' and 'trace'",
		Value: "mem",
	},
}

var adminProfilingStartCmd = cli.Command{
	Name:            "start",
	Usage:           "Start recording profiling data",
	Action:          mainAdminProfilingStart,
	Before:          setGlobalsFromContext,
	Flags:           append(adminProfilingStartFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Start CPU profiling
       $ {{.HelpName}} --type cpu myminio/

`,
}

func checkAdminProfilingStartSyntax(ctx *cli.Context) {
	// Check flags combinations
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "start", 1) // last argument is exit code
	}

	profilerTypes := []madmin.ProfilerType{
		madmin.ProfilerCPU,
		madmin.ProfilerMEM,
		madmin.ProfilerBlock,
		madmin.ProfilerMutex,
		madmin.ProfilerTrace,
	}

	// Check if the provided profiler type is known and supported
	supportedProfiler := false
	profilerType := strings.ToLower(ctx.String("type"))
	for _, profiler := range profilerTypes {
		if profilerType == string(profiler) {
			supportedProfiler = true
			break
		}
	}
	if !supportedProfiler {
		fatalIf(errDummy(), "Profiler type unrecognized. Possible values are: %v.", profilerTypes)
	}
}

// mainAdminProfilingStart - the entry function of profiling command
func mainAdminProfilingStart(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfilingStartSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	profilerType := ctx.String("type")

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Cannot initialize admin client.")
		return nil
	}

	// Start profiling
	_, cmdErr := client.StartProfiling(madmin.ProfilerType(profilerType))
	fatalIf(probe.NewError(cmdErr), "Unable to start profiling.")

	console.Infoln("Profiling data successfully started.")
	return nil
}
