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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminBucketRemoteRmFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "ARN to be removed",
	},
}
var adminBucketRemoteRmCmd = cli.Command{
	Name:         "rm",
	Usage:        "remove configured remote target",
	Action:       mainAdminBucketRemoteRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminBucketRemoteRmFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove existing remote target with arn "arn:minio:replication:us-west-1:993bc6b6-accd-45e3-884f-5f3e652aed2a:dest1"
     for bucket srcbucket on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/srcbucket --arn "arn:minio:replication:us-west-1:993bc6b6-accd-45e3-884f-5f3e652aed2a:dest1"
`,
}

// checkAdminBucketRemoteRemoveSyntax - validate all the passed arguments
func checkAdminBucketRemoteRemoveSyntax(ctx *cli.Context) {

	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
}

// mainAdminBucketRemoteRemove is the handle for "mc admin bucket remote rm" command.
func mainAdminBucketRemoteRemove(ctx *cli.Context) error {
	checkAdminBucketRemoteRemoveSyntax(ctx)

	console.SetColor("RemoteMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr.Trace(aliasedURL), "Unable to initialize admin connection.")
	_, sourceBucket := url2Alias(args[0])
	if sourceBucket == "" {
		fatalIf(errInvalidArgument(), "Source bucket not specified in `"+args[0]+"`.")
	}
	arn := ctx.String("arn")
	if arn == "" {
		fatalIf(errInvalidArgument(), "ARN needs to be specified.")
	}
	fatalIf(probe.NewError(client.RemoveRemoteTarget(globalContext, sourceBucket, arn)).Trace(args...), "Unable to remove remote target")

	printMsg(RemoteMessage{
		op:           ctx.Command.Name,
		SourceBucket: sourceBucket,
		RemoteARN:    arn,
	})

	return nil
}
