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
	"path"
	"strconv"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio/pkg/console"
)

var adminBucketRemoteEditFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "ARN of target",
	}}
var adminBucketRemoteEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "edit credentials for existing remote target",
	Action:       mainAdminBucketRemoteEdit,
	Before:       setGlobalsFromContext,
	OnUsageError: onUsageError,
	Flags:        append(globalFlags, adminBucketRemoteEditFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET http(s)://ACCESSKEY:SECRETKEY@DEST_URL/DEST_BUCKET --arn arn

TARGET:
  Also called as alias/sourcebucketname

DEST_BUCKET:
  Also called as remote target bucket.

DEST_URL:
  Also called as remote endpoint.

ACCESSKEY:
  Also called as username.

SECRETKEY:
  Also called as password.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Edit credentials for existing remote target with arn where a remote target has been configured between sourcebucket on sitea to targetbucket on siteb.
  	{{.DisableHistory}}
  	{{.Prompt}} {{.HelpName}} sitea/sourcebucket \
                 https://foobar:newpassword@minio.siteb.example.com/targetbucket \
                 --arn "arn:minio:replication:us-west-1:993bc6b6-accd-45e3-884f-5f3e652aed2a:dest1"
	{{.EnableHistory}}
`,
}

// checkAdminBucketRemoteEditSyntax - validate all the passed arguments
func checkAdminBucketRemoteEditSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr != 2 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
}

// fetchRemoteEditTarget - returns the dest bucket, dest endpoint, access and secret key
func fetchRemoteEditTarget(cli *cli.Context) (bktTarget *madmin.BucketTarget) {
	args := cli.Args()
	_, sourceBucket := url2Alias(args[0])

	tgtURL := args[1]
	accessKey, secretKey, u := extractCredentialURL(tgtURL)
	var tgtBucket string
	if u.Path != "" {
		tgtBucket = path.Clean(u.Path[1:])
	}
	if e := s3utils.CheckValidBucketName(tgtBucket); e != nil {
		fatalIf(probe.NewError(e).Trace(tgtURL), "Invalid target bucket specified")
	}

	secure := u.Scheme == "https"
	host := u.Host
	if u.Port() == "" {
		port := 80
		if secure {
			port = 443
		}
		host = host + ":" + strconv.Itoa(port)
	}
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	creds := &madmin.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	bktTarget = &madmin.BucketTarget{
		SourceBucket: sourceBucket,
		TargetBucket: tgtBucket,
		Secure:       secure,
		Credentials:  creds,
		Endpoint:     host,
		API:          "s3v4",
		Region:       cli.String("region"),
		Arn:          cli.String("arn"),
	}
	return bktTarget
}

// mainAdminBucketRemoteEdit is the handle for "mc admin bucket remote edit" command.
func mainAdminBucketRemoteEdit(ctx *cli.Context) error {
	checkAdminBucketRemoteEditSyntax(ctx)

	console.SetColor("RemoteMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	bktTarget := fetchRemoteEditTarget(ctx)

	targets, e := client.ListRemoteTargets(globalContext, bktTarget.SourceBucket, "")
	fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch remote target.")

	var found bool
	for _, t := range targets {
		if t.Arn == bktTarget.Arn {
			if t.Endpoint != bktTarget.Endpoint {
				fatalIf(errInvalidArgument().Trace(args...), "configured Endpoint `"+t.Endpoint+"` does not match "+bktTarget.Endpoint+"` for this ARN `"+bktTarget.Arn+"`")
			}
			if t.TargetBucket != bktTarget.TargetBucket {
				fatalIf(errInvalidArgument().Trace(args...), "configured remote target bucket `"+t.TargetBucket+"` does not match "+bktTarget.TargetBucket+"` for this ARN `"+bktTarget.Arn+"`")
			}
			if t.SourceBucket != bktTarget.SourceBucket {
				fatalIf(errInvalidArgument().Trace(args...), "configured source bucket `"+t.SourceBucket+"` does not match "+bktTarget.SourceBucket+"` for this ARN `"+bktTarget.Arn+"`")
			}
			found = true
			break
		}
	}
	if !found {
		fatalIf(errInvalidArgument().Trace(args...), "Unable to edit remote target - `"+bktTarget.Arn+"` is not a valid Arn")
	}
	arn, e := client.UpdateRemoteTarget(globalContext, bktTarget)
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to update credentials for remote target `"+bktTarget.Endpoint+"` from `"+bktTarget.SourceBucket+"` -> `"+bktTarget.TargetBucket+"`")
	}

	printMsg(RemoteMessage{
		op:           ctx.Command.Name,
		TargetURL:    bktTarget.URL().String(),
		TargetBucket: bktTarget.TargetBucket,
		AccessKey:    bktTarget.Credentials.AccessKey,
		SourceBucket: bktTarget.SourceBucket,
		RemoteARN:    arn,
	})

	return nil
}
