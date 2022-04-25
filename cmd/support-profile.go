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
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/console"
)

// profile command flags.
var (
	profileFlags = []cli.Flag{
		cli.IntFlag{
			Name:  "count",
			Usage: "number of times the profiling is needed",
			Value: 1,
		},
		cli.IntFlag{
			Name:  "interval",
			Usage: "interval between two profile data in seconds",
			Value: 30,
		},
		cli.StringFlag{
			Name:  "type",
			Usage: "profiler type, possible values are 'cpu', 'cpuio', 'mem', 'block', 'mutex', 'trace', 'threads' and 'goroutines'",
			Value: "cpu,mem,block,mutex,threads,goroutines",
		},
	}
)

var supportProfileSubcommands = []cli.Command{
	supportProfileStartCmd,
	supportProfileStopCmd,
}

var supportProfileCmd = cli.Command{
	Name:            "profile",
	Usage:           "generate profile data for debugging",
	Action:          mainSupportProfile,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(profileFlags, globalFlags...),
	Subcommands:     supportProfileSubcommands,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Fetch CPU profiling only
       {{.Prompt}} {{.HelpName}} --type cpu myminio/

    2. Fetch CPU, Memory and Block profiling concurrently
       {{.Prompt}} {{.HelpName}} --type cpu,mem,block myminio/

	3. Fetch CPU, Memory and Block profiling concurrently 3 times
       {{.Prompt}} {{.HelpName}} --type cpu,mem,block --count 3 myminio/ 

	4. Fetch CPU, Memory and Block profiling concurrently 2 times with an interval of 10 mins
       {{.Prompt}} {{.HelpName}} --type cpu,mem,block  --count 2 --interval 600 myminio/

`,
}

func checkAdminProfileSyntax(ctx *cli.Context) {
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

	var interval int
	if ctx.IsSet("interval") {
		interval = ctx.Int("interval")
		if interval <= 30 {
			fatalIf(errInvalidArgument().Trace(ctx.Args()...), " the minimum interval between two profiling must be 30 seconds, for example: '--interval 30 --count 2' to get two profilers output at 30 seconds interval")
		}
		if interval >= 30 && (ctx.Int("count") <= 1) {
			fatal(errDummy().Trace(), "count flag must be specified with value greater than 1 when '--interval' flag is specified.")
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

func getProfileData(client *madmin.AdminClient) string {
	// Create profile zip file
	tmpFile, e := ioutil.TempFile("", "mc-profile-")
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Ask for profile data, which will come compressed with zip format
	zippedData, adminErr := client.DownloadProfilingData(globalContext)
	fatalIf(probe.NewError(adminErr), "Unable to download profile data.")

	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, zippedData)
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Close everything
	zippedData.Close()
	tmpFile.Close()

	downloadPath := "profile.zip"
	downloadedFile := downloadPath + "." + time.Now().Format(dateTimeFormatFilename)

	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = moveFile(downloadPath, downloadedFile)
		fatalIf(probe.NewError(e), "Unable to create a backup of profile.zip")
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to download profile data.")
		}
	}
	fatalIf(probe.NewError(moveFile(tmpFile.Name(), downloadPath)), "Unable to download profile data.")
	return downloadedFile
}

// mainSupportProfile is the handle for "mc support profile" command.
func mainSupportProfile(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfileSyntax(ctx)
	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	profilers := ctx.String("type")
	count := ctx.Int("count")
	interval := ctx.Int("interval")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	for i := 1; i <= count; i++ {
		_, cmdErr := client.StartProfiling(globalContext, madmin.ProfilerType(profilers))
		fatalIf(probe.NewError(cmdErr), "Unable to start profile. ")
		console.Infof("count %d : Profile data successfully started. \n", i)

		log.Println("Collecting data for 10 seconds")
		sleep := time.Duration(10)
		time.Sleep(time.Second * sleep)
		log.Println("Stopping profiling..")

		getProfileData(client)
		console.Infof("count %d : Profile data successfully downloaded as %s\n", i, getProfileData(client))

		if count > 1 && i <= count-1 {
			log.Printf("Waiting for %d seconds, before starting another profile\n", interval)
			sleepBetweenIntervals := time.Duration(interval)
			time.Sleep(time.Second * sleepBetweenIntervals)
		}
	}
	return nil
}
