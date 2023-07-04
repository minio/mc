// Copyright (c) 2015-2022 MinIO, Inc.
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
	"os"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/console"
)

// profile command flags.
var (
	profileFlags = append([]cli.Flag{
		cli.IntFlag{
			Name:  "duration",
			Usage: "profile for the specified duration in seconds",
			Value: 10,
		},
		cli.StringFlag{
			Name:  "type",
			Usage: "profiler type, possible values are 'cpu', 'cpuio', 'mem', 'block', 'mutex', 'trace', 'threads' and 'goroutines'",
			Value: "cpu,mem,block,mutex,goroutines",
		},
	}, subnetCommonFlags...)
)

const profileFile = "profile.zip"

var supportProfileCmd = cli.Command{
	Name:            "profile",
	Usage:           "upload profile data for debugging",
	Action:          mainSupportProfile,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(profileFlags, supportGlobalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Profile CPU for 10 seconds on cluster with alias 'myminio' and upload results to SUBNET
     {{.Prompt}} {{.HelpName}} --type cpu myminio

  2. Profile CPU, Memory, Goroutines for 10 seconds on cluster with alias 'myminio' and upload results to SUBNET
     {{.Prompt}} {{.HelpName}} --type cpu,mem,goroutines myminio

  3. Profile CPU, Memory, Goroutines for 10 minutes on cluster with alias 'myminio' and upload results to SUBNET
     {{.Prompt}} {{.HelpName}} --type cpu,mem,goroutines --duration 600 myminio

  4. Profile CPU for 10 seconds on cluster with alias 'myminio', save and upload to SUBNET manually
     {{.Prompt}} {{.HelpName}} --type cpu --airgap myminio
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
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
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
	inputFile, e := os.Open(sourcePath)
	if e != nil {
		return e
	}

	outputFile, e := os.Create(destPath)
	if e != nil {
		inputFile.Close()
		return e
	}
	defer outputFile.Close()

	if _, e = io.Copy(outputFile, inputFile); e != nil {
		inputFile.Close()
		return e
	}

	// The copy was successful, so now delete the original file
	inputFile.Close()
	return os.Remove(sourcePath)
}

func saveProfileFile(data io.ReadCloser) {
	// Create profile zip file
	tmpFile, e := os.CreateTemp("", "mc-profile-")
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, data)
	fatalIf(probe.NewError(e), "Unable to download profile data.")

	// Close everything
	data.Close()
	tmpFile.Close()

	downloadedFile := profileFile + "." + time.Now().Format(dateTimeFormatFilename)

	fi, e := os.Stat(profileFile)
	if e == nil && !fi.IsDir() {
		e = moveFile(profileFile, downloadedFile)
		fatalIf(probe.NewError(e), "Unable to create a backup of profile.zip")
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to save profile data")
		}
	}
	fatalIf(probe.NewError(moveFile(tmpFile.Name(), profileFile)), "Unable to save profile data")
}

// mainSupportProfile is the handle for "mc support profile" command.
func mainSupportProfile(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfileSyntax(ctx)

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)
	alias, apiKey := initSubnetConnectivity(ctx, aliasedURL, true)
	if len(apiKey) == 0 {
		// api key not passed as flag. Check that the cluster is registered.
		apiKey = validateClusterRegistered(alias, true)
	}

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Main execution
	execSupportProfile(ctx, client, alias, apiKey)
	return nil
}

func execSupportProfile(ctx *cli.Context, client *madmin.AdminClient, alias string, apiKey string) {
	var reqURL string
	var headers map[string]string
	profilers := ctx.String("type")
	duration := ctx.Int("duration")

	if !globalAirgapped {
		// Retrieve subnet credentials (login/license) beforehand as
		// it can take a long time to fetch the profile data
		uploadURL := subnetUploadURL("profile", profileFile)
		reqURL, headers = prepareSubnetUploadURL(uploadURL, alias, apiKey)
	}

	console.Infof("Profiling '%s' for %d seconds... \n", alias, duration)
	data, e := client.Profile(globalContext, madmin.ProfilerType(profilers), time.Second*time.Duration(duration))
	fatalIf(probe.NewError(e), "Unable to save profile data")

	saveProfileFile(data)

	if !globalAirgapped {
		_, e = uploadFileToSubnet(alias, profileFile, reqURL, headers)
		if e != nil {
			errorIf(probe.NewError(e), "Unable to upload profile file to SUBNET")
			console.Infof("Profiling data saved locally at '%s'\n", profileFile)
			return
		}
		console.Infoln("Profiling data uploaded to SUBNET successfully")
	} else {
		console.Infoln("Profiling data saved successfully at", profileFile)
	}
}
