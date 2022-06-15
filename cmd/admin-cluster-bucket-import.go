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

	"github.com/klauspost/compress/zip"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminClusterBucketImportCmd = cli.Command{
	Name:            "import",
	Usage:           "restore bucket metadata from a zip file",
	Action:          mainClusterBucketImport,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET/[BUCKET] /path/to/backups/bucket-metadata.zip

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set bucket metadata for 'mybucket' from previously exported metadata zip file.
     {{.Prompt}} {{.HelpName}} myminio/mybucket /backups/mybucket-metadata.zip

  2. Set bucket metadata for all buckets from previously exported metadata zip file.
     {{.Prompt}} {{.HelpName}} myminio /backups/cluster-bucket-metadata.zip
`,
}

func checkBucketImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "import", 1) // last argument is exit code
	}
}

// mainClusterBucketImport - bucket metadata import command
func mainClusterBucketImport(ctx *cli.Context) error {
	// Check for command syntax
	checkBucketImportSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	var r io.Reader
	var sz int64
	f, e := os.Open(args.Get(1))
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get bucket metadata")
	}
	if st, e := f.Stat(); e == nil {
		sz = st.Size()
	}
	defer f.Close()
	r = f

	_, e = zip.NewReader(r.(io.ReaderAt), sz)
	fatalIf(probe.NewError(e).Trace(args...), fmt.Sprintf("Unable to read zip file %s", args.Get(1)))

	f, e = os.Open(args.Get(1))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get bucket metadata")

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

	ierr := client.ImportBucketMetadata(context.Background(), bucket, f)
	fatalIf(probe.NewError(ierr).Trace(aliasedURL), "Unable to import bucket metadata.")
	if !globalJSON {
		console.Infof("Bucket metadata imported to %s from %s\n", bucket, args.Get(1))
	}
	return nil
}
