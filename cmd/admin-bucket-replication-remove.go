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

var adminBucketReplicationRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove configured replication target",
	Action: mainAdminBucketReplicationRemove,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove existing replication target for bucket srcbucket on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/srcbucket
`,
}

// checkAdminBucketReplicationRemoveSyntax - validate all the passed arguments
func checkAdminBucketReplicationRemoveSyntax(ctx *cli.Context) {

	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
}

// mainAdminBucketReplicationRemove is the handle for "mc admin bucket replication remove" command.
func mainAdminBucketReplicationRemove(ctx *cli.Context) error {
	checkAdminBucketReplicationRemoveSyntax(ctx)

	console.SetColor("ReplicationMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	_, sourceBucket := url2Alias(args[0])
	fatalIf(probe.NewError(client.SetBucketReplicationTarget(globalContext, sourceBucket, nil)).Trace(args...), "Cannot remove replication target")

	printMsg(replicationMessage{
		op:           "remove",
		SourceBucket: sourceBucket,
	})

	return nil
}
