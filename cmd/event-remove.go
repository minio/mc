/*
 * MinIO Client (C) 2016 MinIO, Inc.
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
	"errors"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	eventRemoveFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "force removing all bucket notifications",
		},
	}
)

var eventRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove a bucket notification; '--force' removes all bucket notifications",
	Action: mainEventRemove,
	Before: setGlobalsFromContext,
	Flags:  append(eventRemoveFlags, globalFlags...),
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

// checkEventRemoveSyntax - validate all the passed arguments
func checkEventRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "remove", 1) // last argument is exit code
	}
	if len(ctx.Args()) == 1 && !ctx.Bool("force") {
		fatalIf(probe.NewError(errors.New("")), "--force flag needs to be passed to remove all bucket notifications.")
	}
}

// eventRemoveMessage container
type eventRemoveMessage struct {
	ARN    string `json:"arn"`
	Status string `json:"status"`
}

// JSON jsonified remove message.
func (u eventRemoveMessage) JSON() string {
	u.Status = "success"
	eventRemoveMessageJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(eventRemoveMessageJSONBytes)
}

func (u eventRemoveMessage) String() string {
	msg := console.Colorize("Event", "Successfully removed "+u.ARN)
	return msg
}

func mainEventRemove(ctx *cli.Context) error {
	console.SetColor("Event", color.New(color.FgGreen, color.Bold))

	checkEventRemoveSyntax(ctx)

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
	printMsg(eventRemoveMessage{ARN: arn})

	return nil
}
