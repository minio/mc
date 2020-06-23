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

var adminBucketReplicationGetCmd = cli.Command{
	Name:   "get",
	Usage:  "get bucket replication target",
	Action: mainAdminBucketReplicationGet,
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
  1. Get replication target on MinIO server for bucket srcbucket.
     {{.Prompt}} {{.HelpName}} myminio/srcbucket
`,
}

// checkAdminBucketReplicationGetSyntax - validate all the passed arguments
func checkAdminBucketReplicationGetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "get", 1) // last argument is exit code
	}
}

// mainAdminBucketReplicationGet is the handle for "mc admin bucket replication get" command.
func mainAdminBucketReplicationGet(ctx *cli.Context) error {
	checkAdminBucketReplicationGetSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("ReplicationMessage", color.New(color.FgGreen))
	console.SetColor("AccessKey", color.New(color.FgBlue))
	console.SetColor("SourceBucket", color.New(color.FgYellow))
	console.SetColor("ReplicaBucket", color.New(color.FgYellow))
	console.SetColor("ReplicaURL", color.New(color.FgHiWhite))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	_, sourceBucket := url2Alias(aliasedURL)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	target, e := client.GetBucketReplicationTarget(globalContext, sourceBucket)
	fatalIf(probe.NewError(e).Trace(args...), "Cannot get replication target")
	printMsg(replicationMessage{
		op:             "get",
		AccessKey:      target.Credentials.AccessKey,
		ReplicaBucket:  target.TargetBucket,
		ReplicaURL:     target.URL(),
		SourceBucket:   sourceBucket,
		ReplicationARN: target.Arn,
	})
	return nil
}
