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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminBucketRemoteListFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "service",
		Usage: "Type of service. Valid options are '[replication]'",
	},
}

var adminBucketRemoteListCmd = cli.Command{
	Name:   "ls",
	Usage:  "list remote target ARN(s)",
	Action: mainAdminBucketRemoteList,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, adminBucketRemoteListFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get remote bucket target for replication on MinIO server for bucket 'srcbucket'.
     {{.Prompt}} {{.HelpName}} myminio/srcbucket --service "replication"

  2. List all remote bucket target(s) on MinIO server for bucket 'srcbucket'.
     {{.Prompt}} {{.HelpName}} myminio/srcbucket

  3. List all remote bucket target(s) on MinIO tenant.
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminBucketRemoteListSyntax - validate all the passed arguments
func checkAdminBucketRemoteListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
}

// mainAdminBucketRemoteList is the handle for "mc admin bucket remote list" command.
func mainAdminBucketRemoteList(ctx *cli.Context) error {
	checkAdminBucketRemoteListSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("RemoteListMessage", color.New(color.Bold, color.FgHiGreen))
	console.SetColor("RemoteListEmpty", color.New(color.FgRed))
	console.SetColor("SourceBucket", color.New(color.FgYellow))
	console.SetColor("TargetBucket", color.New(color.FgYellow))
	console.SetColor("TargetURL", color.New(color.FgHiWhite))
	console.SetColor("ARN", color.New(color.FgCyan))
	console.SetColor("Arrow", color.New(color.FgHiWhite))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	_, sourceBucket := url2Alias(aliasedURL)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")
	svcType := ctx.String("service")
	targets, e := client.ListRemoteTargets(globalContext, sourceBucket, svcType)
	fatalIf(probe.NewError(e).Trace(args...), "Cannot list remote target(s)")
	printRemotes(aliasedURL, targets)
	return nil
}

func printRemotes(urlStr string, targets []madmin.BucketTarget) {

	maxURLLen := 10
	maxTgtLen := 6
	maxSrcLen := 6

	if !globalJSON {
		if len(targets) == 0 {
			console.Print(console.Colorize("RemoteListEmpty", fmt.Sprintf("No remote targets found for`%s`. \n", urlStr)))
			return
		}
		for _, t := range targets {
			l := len(t.Endpoint)
			if l > maxURLLen {
				maxURLLen = l
			}
			if len(t.TargetBucket) > maxTgtLen {
				maxTgtLen = len(t.TargetBucket)
			}
			if len(t.SourceBucket) > maxSrcLen {
				maxSrcLen = len(t.SourceBucket)
			}
		}
		if maxURLLen > 0 {
			console.Println(console.Colorize("RemoteListMessage", fmt.Sprintf("%-*.*s %-*.*s->%-*.*s %s", maxURLLen+8, maxURLLen+8, "Remote URL", maxSrcLen, maxSrcLen, "Source", maxTgtLen, maxTgtLen, "Target", "ARN")))
		}

	}
	for _, target := range targets {
		targetURL := target.URL()
		if !globalJSON {
			if maxURLLen > 0 {
				targetURL = fmt.Sprintf("%-*.*s", maxURLLen+8, maxURLLen+8, target.URL())
			}
			if maxTgtLen > 0 {
				target.TargetBucket = fmt.Sprintf("%-*.*s", maxTgtLen, maxTgtLen, target.TargetBucket)
			}

			if maxSrcLen > 0 {
				target.SourceBucket = fmt.Sprintf("%-*.*s", maxSrcLen, maxSrcLen, target.SourceBucket)
			}
		}

		printMsg(RemoteMessage{
			op:           "ls",
			AccessKey:    target.Credentials.AccessKey,
			TargetBucket: target.TargetBucket,
			TargetURL:    targetURL,
			SourceBucket: target.SourceBucket,
			RemoteARN:    target.Arn,
			ServiceType:  string(target.Type),
		})
	}
}
