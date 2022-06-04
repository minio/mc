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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/cli"
)

var adminClusterDataListCmd = cli.Command{
	Name:            "list",
	Usage:           "create a listing of object (versions) in a bucket prefix",
	Action:          mainClusterDataList,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET/[BUCKET]/[PREFIX] /path/to/list-dir

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a listing of objects in prefix "prefix" of 'mybucket' to /tmp/data.
     {{.Prompt}} {{.HelpName}} myminio/mybucket/prefix /tmp/data
`,
}

const (
	objListing = "obj_listing.csv"
	dmListing  = "dm_listing.csv"
)

func checkDataListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// mainClusterDataList - creates a listing of all object versions in a bucket prefix
func mainClusterDataList(ctx *cli.Context) error {
	// Check for command syntax
	checkDataListSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	clnt, e := newClient(aliasedURL)
	if e != nil {
		fatalIf(e.Trace(aliasedURL), "Unable to initialize target `"+aliasedURL+"`.")
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, _ := splits[1], splits[2]
	if bucket == "" {
		fatalIf(errInvalidArgument().Trace(args.Get(0)), "Bucket not specified")
	}
	f, err := os.OpenFile(path.Join(args.Get(1), objListing), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatalln("Could not open file path", path.Join(args.Get(1), objListing), err)
	}
	defer f.Close()
	dataWriter := bufio.NewWriter(f)
	defer dataWriter.Flush()

	df, ferr := os.OpenFile(path.Join(args.Get(1), dmListing), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if ferr != nil {
		log.Fatalln("Could not open file path", path.Join(args.Get(1), dmListing), ferr)
	}
	defer df.Close()
	dmDataWriter := bufio.NewWriter(df)
	defer dmDataWriter.Flush()

	// List all objects from a bucket-name with a matching prefix.
	for c := range clnt.List(context.Background(), ListOptions{
		Recursive:         true,
		WithDeleteMarkers: true,
		WithOlderVersions: true,
		WithMetadata:      true,
	}) {
		if c.Err != nil {
			log.Fatalln("LIST error:", c.Err)
			continue
		}
		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(c.URL.Path)
		// Trim prefix from the content path.
		c.URL.Path = strings.TrimPrefix(contentURL, slashSeperator)
		c.URL.Path = strings.TrimPrefix(c.URL.Path, bucket)
		key := strings.TrimPrefix(c.URL.Path, slashSeperator)

		str := fmt.Sprintf("%s, %s, %s, %t\n", c.BucketName, key, c.VersionID, c.IsDeleteMarker)
		switch c.IsDeleteMarker {
		case true:
			if _, err := dmDataWriter.WriteString(str); err != nil {
				log.Println("Error writing delete marker to file:", c.BucketName, getKey(c), c.VersionID, err)
			}
		default:
			if _, err := dataWriter.WriteString(str); err != nil {
				log.Println("Error writing object to file:", c.BucketName, getKey(c), c.VersionID, err)
			}
		}
	}
	return nil
}
