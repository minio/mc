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

package cmd

import (
	"encoding/json"
	"errors"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	eventsRemoveFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force removing all bucket notifications",
		},
	}
)

var eventsRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "Remove a bucket notification. With '--force' can remove all bucket notifications.",
	Action: mainEventsRemove,
	Before: setGlobalsFromContext,
	Flags:  append(eventsRemoveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [ARN] [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Remove bucket notification associated to a specific arn
     $ {{.HelpName}} myminio/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue

   2. Remove all bucket notifications. --force flag is mandatory here
     $ {{.HelpName}} myminio/mybucket --force

`,
}

// checkEventsRemoveSyntax - validate all the passed arguments
func checkEventsRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
	if len(ctx.Args()) == 1 && !ctx.Bool("force") {
		fatalIf(probe.NewError(errors.New("")), "--force flag needs to be passed to remove all bucket notifications.")
	}
}

// eventsRemoveMessage container
type eventsRemoveMessage struct {
	ARN    string `json:"arn"`
	Status string `json:"status"`
}

// JSON jsonified remove message.
func (u eventsRemoveMessage) JSON() string {
	u.Status = "success"
	eventsRemoveMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(eventsRemoveMessageJSONBytes)
}

func (u eventsRemoveMessage) String() string {
	msg := console.Colorize("Events", "Successfully removed "+u.ARN)
	return msg
}

func mainEventsRemove(ctx *cli.Context) error {
	console.SetColor("Events", color.New(color.FgGreen, color.Bold))

	checkEventsRemoveSyntax(ctx)

	args := ctx.Args()
	path := args.Get(0)

	arn := ""
	if len(args) == 2 {
		arn = args.Get(1)
	}

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	s3Client, ok := client.(*s3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	err = s3Client.RemoveNotificationConfig(arn)
	fatalIf(err, "Cannot disable notification on the specified bucket.")
	printMsg(eventsRemoveMessage{ARN: arn})

	return nil
}
