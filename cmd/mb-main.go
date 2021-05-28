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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var (
	mbFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "region",
			Value: "us-east-1",
			Usage: "specify bucket region; defaults to 'us-east-1'",
		},
		cli.BoolFlag{
			Name:  "ignore-existing, p",
			Usage: "ignore if bucket/directory already exists",
		},
		cli.BoolFlag{
			Name:  "with-lock, l",
			Usage: "enable object lock",
		},
	}
)

// make a bucket.
var mbCmd = cli.Command{
	Name:         "mb",
	Usage:        "make a bucket",
	Action:       mainMakeBucket,
	Before:       setGlobalsFromContext,
	OnUsageError: onUsageError,
	Flags:        append(mbFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
  1. Create a bucket on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} s3/mynewbucket

  2. Create a new bucket on Google Cloud Storage.
     {{.Prompt}} {{.HelpName}} gcs/miniocloud

  3. Create a new bucket on Amazon S3 cloud storage in region 'us-west-2'.
     {{.Prompt}} {{.HelpName}} --region=us-west-2 s3/myregionbucket

  4. Create a new directory including its missing parents (equivalent to 'mkdir -p').
     {{.Prompt}} {{.HelpName}} /tmp/this/new/dir1

  5. Create multiple directories including its missing parents (behavior similar to 'mkdir -p').
     {{.Prompt}} {{.HelpName}} /mnt/sdb/mydisk /mnt/sdc/mydisk /mnt/sdd/mydisk

  6. Ignore if bucket/directory already exists.
     {{.Prompt}} {{.HelpName}} --ignore-existing myminio/mynewbucket

  7. Create a new bucket on Amazon S3 cloud storage in region 'us-west-2' with object lock enabled.
     {{.Prompt}} {{.HelpName}} --with-lock --region=us-west-2 s3/myregionbucket
`,
}

// makeBucketMessage is container for make bucket success and failure messages.
type makeBucketMessage struct {
	Status string `json:"status"`
	Bucket string `json:"bucket"`
	Region string `json:"region"`
}

// String colorized make bucket message.
func (s makeBucketMessage) String() string {
	return console.Colorize("MakeBucket", "Bucket created successfully `"+s.Bucket+"`.")
}

// JSON jsonified make bucket message.
func (s makeBucketMessage) JSON() string {
	makeBucketJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(makeBucketJSONBytes)
}

// Validate command line arguments.
func checkMakeBucketSyntax(cliCtx *cli.Context) {
	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "mb", 1) // last argument is exit code
	}
}

// mainMakeBucket is entry point for mb command.
func mainMakeBucket(cli *cli.Context) error {

	// check 'mb' cli arguments.
	checkMakeBucketSyntax(cli)

	// Additional command speific theme customization.
	console.SetColor("MakeBucket", color.New(color.FgGreen, color.Bold))

	// Save region.
	region := cli.String("region")
	ignoreExisting := cli.Bool("p")
	withLock := cli.Bool("l")

	var cErr error
	for _, targetURL := range cli.Args() {
		// Instantiate client for URL.
		clnt, err := newClient(targetURL)
		if err != nil {
			errorIf(err.Trace(targetURL), "Invalid target `"+targetURL+"`.")
			cErr = exitStatus(globalErrorExitStatus)
			continue
		}

		ctx, cancelMakeBucket := context.WithCancel(globalContext)
		defer cancelMakeBucket()

		// Make bucket.
		err = clnt.MakeBucket(ctx, region, ignoreExisting, withLock)
		if err != nil {
			switch err.ToGoError().(type) {
			case BucketNameEmpty:
				errorIf(err.Trace(targetURL), "Unable to make bucket, please use `mc mb %s/<your-bucket-name>`.", targetURL)
			case BucketNameTopLevel:
				errorIf(err.Trace(targetURL), "Unable to make prefix, please use `mc mb %s/`.", targetURL)
			default:
				errorIf(err.Trace(targetURL), "Unable to make bucket `"+targetURL+"`.")
			}
			cErr = exitStatus(globalErrorExitStatus)
			continue
		}

		// Successfully created a bucket.
		printMsg(makeBucketMessage{Status: "success", Bucket: targetURL})
	}
	return cErr
}
