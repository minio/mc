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
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var adminProfilingStopCmd = cli.Command{
	Name:            "stop",
	Usage:           "Stop and download profiling data",
	Action:          mainAdminProfilingStop,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    2. Download latest profiling data in the current directory
       $ {{.HelpName}} myminio/
`,
}

func checkAdminProfilingStopSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "stop", 1) // last argument is exit code
	}
}

// mainAdminProfilingStop - the entry function of profiling stop command
func mainAdminProfilingStop(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfilingStopSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Cannot initialize admin client.")
		return nil
	}

	// Create profiling zip file
	tmpFile, e := ioutil.TempFile("", "mc-profiling-")
	fatalIf(probe.NewError(e), "Unable to download profiling data.")

	// Ask for profiling data, which will come compressed with zip format
	zippedData, adminErr := client.DownloadProfilingData()
	fatalIf(probe.NewError(adminErr), "Unable to download profiling data.")

	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, zippedData)
	fatalIf(probe.NewError(e), "Unable to download profiling data.")

	// Close everything
	zippedData.Close()
	tmpFile.Close()

	downloadPath := "profiling.zip"

	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = os.Rename(downloadPath, downloadPath+"."+time.Now().Format("2006-01-02T15:04:05.999999-07:00"))
		fatalIf(probe.NewError(e), "Unable to create a backup of profiling.zip")
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to download profiling data.")
		}
	}

	fatalIf(probe.NewError(os.Rename(tmpFile.Name(), downloadPath)), "Unable to download profiling data.")

	console.Infof("Profiling data successfully downloaded as %s\n", downloadPath)
	return nil
}
