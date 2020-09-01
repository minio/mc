/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminBucketRemoteRmFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "ARN to be removed",
	},
}
var adminBucketRemoteRmCmd = cli.Command{
	Name:   "rm",
	Usage:  "remove configured remote target",
	Action: mainAdminBucketRemoteRemove,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, adminBucketRemoteRmFlags...),
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
