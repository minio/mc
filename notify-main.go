/*
 * Minio Client (C) 2016 Minio, Inc.
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

package main

import (
	"fmt"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/probe"
)

var (
	notifyFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "account-region",
			Value: "us-east-1",
			Usage: "Specify notification region. Defaults to ‘us-east-1’.",
		},
		cli.StringFlag{
			Name:  "account-id",
			Value: "mc",
			Usage: "Specify notification account id. Defaults to ‘mc’.",
		},
	}
)

var notifyCmd = cli.Command{
	Name:   "notify",
	Usage:  "Print realtime bucket notification.",
	Action: mainNotify,
	Flags:  append(notifyFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Watch new S3 operations on a minio server
      $ mc {{.Name}} myminio/testbucket

   2. Watch new events on a specific region and account id
      $ mc {{.Name}} myminio/testbucket --account-region us-west-2 --account-id 81132344
`,
}

// checkNotifySyntax - validate all the passed arguments
func checkNotifySyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "notify", 1) // last argument is exit code
	}
}

func mainNotify(ctx *cli.Context) {

	setGlobalsFromContext(ctx)

	checkNotifySyntax(ctx)

	args := ctx.Args()
	path := args[0]

	region := ctx.String("account-region")
	accountId := ctx.String("account-id")

	client, pErr := newClient(path)
	if pErr != nil {
		fatalIf(pErr.Trace(), "Cannot parse the provided url.")
	}

	// For the moment, we only support s3
	s3Client, ok := client.(*s3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	// Start watching on events
	notificationCh, err := s3Client.Watch(region, accountId, nil)
	fatalIf(probe.NewError(err), "Cannot watch on the specified bucket.")

	// Print all notifications as we receive them
	for notification := range notificationCh {
		if notification.Err != nil {
			// Ignore errors
			continue
		}
		for _, info := range notification.Records {
			fmt.Printf("%s\t%s\t%s\n", info.EventName, info.S3.Bucket.Name, info.S3.Object.Key)
		}
	}
}
