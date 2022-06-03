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
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
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
			Name:  "duration",
			Usage: "start profiling for the specified duration in seconds",
			Value: 10,
		},
		cli.StringFlag{
			Name:  "type",
			Usage: "profiler type, possible values are 'cpu', 'cpuio', 'mem', 'block', 'mutex', 'trace', 'threads' and 'goroutines'",
			Value: "cpu,mem,block,mutex,threads,goroutines",
		},
	}
)

var supportProfileCmd = cli.Command{
	Name:            "profile",
	Usage:           "generate profile data for debugging",
	Action:          mainSupportProfile,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(profileFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Profile CPU for 10 seconds.
     {{.Prompt}} {{.HelpName}} --type cpu myminio/

  2. Profile CPU, Memory, Goroutines for 10 seconds.
     {{.Prompt}} {{.HelpName}} --type cpu,mem,goroutines myminio/

  3. Profile CPU, Memory, Goroutines for 10 minutes.
     {{.Prompt}} {{.HelpName}} --type cpu,mem,goroutines --duration 600 myminio/
`,
}

func checkAdminProfileSyntax(ctx *cli.Context) {
	s := set.CreateStringSet(string(madmin.ProfilerCPU),
		string(madmin.ProfilerMEM),
		string(madmin.ProfilerBlock),
		string(madmin.ProfilerMutex),
		string(madmin.ProfilerTrace),
		string(madmin.ProfilerThreads),
		string(madmin.ProfilerGoroutines),
		string(madmin.ProfilerCPUIO))
	// Check if the provided profiler type is known and supported
	profilers := strings.Split(strings.ToLower(ctx.String("type")), ",")
	for _, profiler := range profilers {
		if profiler != "" {
			if !s.Contains(profiler) {
				fatalIf(errDummy().Trace(ctx.String("type")),
					"Profiler type %s unrecognized. Possible values are: %v.", profiler, s)
			}
		}
	}
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "profile", 1) // last argument is exit code
	}

	if ctx.Int("duration") < 10 {
		fatal(errDummy().Trace(), "profiling must be run for atleast 10 seconds")
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

func getProfileData(data io.ReadCloser) string {
	// Create profile zip file
	tmpFile, e := ioutil.TempFile("", "mc-profile-")
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, data)
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Close everything
	data.Close()
	tmpFile.Close()

	downloadPath := "profile.zip"
	downloadedFile := downloadPath + "." + time.Now().Format(dateTimeFormatFilename)

	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = moveFile(downloadPath, downloadedFile)
		fatalIf(probe.NewError(e), "Unable to create a backup of profile.zip")
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to save profile data")
		}
	}
	fatalIf(probe.NewError(moveFile(tmpFile.Name(), downloadPath)), "Unable to save profile data")
	return downloadPath
}

// mainSupportProfile is the handle for "mc support profile" command.
func mainSupportProfile(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfileSyntax(ctx)
	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	profilers := ctx.String("type")
	duration := ctx.Int("duration")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client")
		return nil
	}

	console.Infof("Profiling '%s' for %d seconds... ", aliasedURL, duration)
	data, adminErr := client.Profile(globalContext, madmin.ProfilerType(profilers), time.Second*time.Duration(duration))

	fatalIf(probe.NewError(adminErr), "Unable to save profile data")
	clr := color.New(color.FgGreen, color.Bold)
	clr.Printf("saved successfully at '%s'\n", getProfileData(data))
	return nil
}
