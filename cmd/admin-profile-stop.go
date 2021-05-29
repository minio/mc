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
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminProfileStopCmd = cli.Command{
	Name:            "stop",
	Usage:           "stop and download profile data",
	Action:          mainAdminProfileStop,
	OnUsageError:    onUsageError,
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
    2. Download latest profile data in the current directory
       {{.Prompt}} {{.HelpName}} myminio/
`,
}

func checkAdminProfileStopSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "stop", 1) // last argument is exit code
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

// mainAdminProfileStop - the entry function of profile stop command
func mainAdminProfileStop(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminProfileStopSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

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

	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = moveFile(downloadPath, downloadPath+"."+time.Now().Format(dateTimeFormatFilename))
		fatalIf(probe.NewError(e), "Unable to create a backup of profile.zip")
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to download profile data.")
		}
	}

	fatalIf(probe.NewError(moveFile(tmpFile.Name(), downloadPath)), "Unable to download profile data.")

	console.Infof("Profile data successfully downloaded as %s\n", downloadPath)
	return nil
}
