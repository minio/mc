// Copyright (c) 2022 MinIO, Inc.
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
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminClusterBucketExportCmd = cli.Command{
	Name:            "export",
	Usage:           "backup bucket metadata to a zip file",
	Action:          mainClusterBucketExport,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET/[BUCKET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Save metadata of all buckets to a zip file.
     {{.Prompt}} {{.HelpName}} myminio
`,
}

func checkBucketExportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainClusterBucketExport -  metadata export command
func mainClusterBucketExport(ctx *cli.Context) error {
	// Check for command syntax
	checkBucketExportSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	console.SetColor("File", color.New(color.FgWhite, color.Bold))

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	aliasedURL = filepath.Clean(aliasedURL)
	_, bucket := url2Alias(aliasedURL)
	r, e := client.ExportBucketMetadata(context.Background(), bucket)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to export bucket metadata.")

	if bucket == "" {
		bucket = "bucket"
	}
	// Create bucket metadata zip file
	tmpFile, e := os.CreateTemp("", fmt.Sprintf("%s-%s-metadata-", strings.ReplaceAll(aliasedURL, "/", "-"), bucket))
	fatalIf(probe.NewError(e), "Unable to download file data.")

	ext := "zip"
	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, r)
	fatalIf(probe.NewError(e), "Unable to download bucket metadata.")
	// Close everything
	r.Close()
	tmpFile.Close()
	// We use 4 bytes of the 32 bytes to identify the file.
	downloadPath := fmt.Sprintf("%s-%s-metadata.%s", aliasedURL, bucket, ext)
	// Create necessary directories.
	dir := filepath.Dir(downloadPath)
	if e := os.MkdirAll(dir, 0o755); e != nil {
		fatalIf(probe.NewError(e).Trace(dir), "Unable to create download directory")
	}

	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = moveFile(downloadPath, downloadPath+"."+time.Now().Format(dateTimeFormatFilename))
		fatalIf(probe.NewError(e), "Unable to create a backup of "+downloadPath)
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to download file data")
		}
	}
	fatalIf(probe.NewError(moveFile(tmpFile.Name(), downloadPath)), "Unable to rename downloaded data, file exists at %s", tmpFile.Name())

	// Explicitly set permissions to 0o600 and override umask
	// to ensure that the file is not world-readable.
	e = os.Chmod(downloadPath, 0o600)
	fatalIf(probe.NewError(e), "Unable to set file permissions for "+downloadPath)

	if !globalJSON {
		console.Infof("Bucket metadata successfully downloaded as %s\n", downloadPath)
		return nil
	}
	v := struct {
		File string `json:"file"`
		Key  string `json:"key,omitempty"`
	}{
		File: downloadPath,
	}
	b, e := json.Marshal(v)
	fatalIf(probe.NewError(e), "Unable to serialize data")
	console.Println(string(b))
	return nil
}
