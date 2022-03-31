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
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/console"
)

var supportProfileFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "type",
		Usage: "start profiler type, possible values are 'cpu', 'cpuio' 'mem', 'block', 'mutex', 'trace', 'threads' and 'goroutines'",
		Value: "cpu,mem,block,goroutines",
	},
	cli.DurationFlag{
		Name:  "duration",
		Usage: "specify duration for profile",
		Value: 60 * time.Second,
	},
}

var supportProfileCmd = cli.Command{
	Name:            "profile",
	Usage:           "record profile data",
	Action:          mainSupportProfile,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportProfileFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Take CPU profile only
       {{.Prompt}} {{.HelpName}} --type cpu myminio/

    2. Take CPU, Memory and Block profile concurrently
       {{.Prompt}} {{.HelpName}} --type cpu,mem,block myminio/

    3. Take all profiles concurrently for a minute
       {{.Prompt}} {{.HelpName}} --duration 60s myminio/
`,
}

func checkAdminProfileSyntax(ctx *cli.Context) {
	// Check flags combinations
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "profile", 1) // last argument is exit code
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
		madmin.ProfilerCPUIO,
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

// moveFile - os.Rename cannot handle cross device renames, in our situation
// it is possible that /tmp is mounted from a separate partition and current
// working directory is a different partition. To allow all situations to
// be handled appropriately use this function instead of os.Rename()
func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return err
	}

	// The copy was successful, so now delete the original file
	return os.Remove(sourcePath)
}

// mainSupportProfile - the entry function of profile command
func mainSupportProfile(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfileSyntax(ctx)

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
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Gathering profile... "
	if !globalJSON {
		s.Start()
	}
	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()
	// Take profile
	zippedData, adminErr := client.Profile(ctxt, madmin.ProfilerType(profilers), ctx.Duration("duration"))
	fatalIf(probe.NewError(adminErr), "Unable to download profile data.")

	// Create profile zip file
	tmpFile, e := ioutil.TempFile("", "mc-profile-")
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, zippedData)
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Close everything
	zippedData.Close()
	tmpFile.Close()

	downloadPath := "profile.zip"

	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = moveFile(downloadPath, downloadPath+"."+time.Now().Format(dateTimeFormatFilename))
		fatalIf(probe.NewError(e), "Unable to create a backup of profile.zip")
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to download profile data.")
		}
	}
	s.Stop()
	fatalIf(probe.NewError(moveFile(tmpFile.Name(), downloadPath)), "Unable to download profile data.")
	console.Infof("Profile data successfully downloaded as %s\n", downloadPath)
	return nil
}
