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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	eventsAddFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "events",
			Value: "put,delete,get",
			Usage: "Filter specific type of events. Defaults to all events.",
		},
		cli.StringFlag{
			Name:  "prefix",
			Usage: "Filter events associated to the specified prefix",
		},
		cli.StringFlag{
			Name:  "suffix",
			Usage: "Filter events associated to the specified suffix",
		},
	}
)

var eventsAddCmd = cli.Command{
	Name:   "add",
	Usage:  "Add a new bucket notification.",
	Action: mainEventsAdd,
	Before: setGlobalsFromContext,
	Flags:  append(eventsAddFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ARN [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Enable bucket notification with a specific arn
     $ {{.HelpName}} myminio/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue

   2. Enable bucket notification with filters parameters
     $ {{.HelpName}} s3/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue --events put,delete,get --prefix photos/ --suffix .jpg

`,
}

// checkEventsAddSyntax - validate all the passed arguments
func checkEventsAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "add", 1) // last argument is exit code
	}
}

// eventsAddMessage container
type eventsAddMessage struct {
	ARN    string   `json:"arn"`
	Events []string `json:"events"`
	Prefix string   `json:"prefix"`
	Suffix string   `json:"suffix"`
	Status string   `json:"status"`
}

// JSON jsonified update message.
func (u eventsAddMessage) JSON() string {
	u.Status = "success"
	eventsAddMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(eventsAddMessageJSONBytes)
}

func (u eventsAddMessage) String() string {
	msg := console.Colorize("Events", "Successfully added "+u.ARN)
	return msg
}

func mainEventsAdd(ctx *cli.Context) error {
	console.SetColor("Events", color.New(color.FgGreen, color.Bold))

	checkEventsAddSyntax(ctx)

	args := ctx.Args()
	path := args[0]
	arn := args[1]

	events := strings.Split(ctx.String("events"), ",")
	prefix := ctx.String("prefix")
	suffix := ctx.String("suffix")

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	s3Client, ok := client.(*s3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	err = s3Client.AddNotificationConfig(arn, events, prefix, suffix)
	fatalIf(err, "Cannot enable notification on the specified bucket.")
	printMsg(eventsAddMessage{
		ARN:    arn,
		Events: events,
		Prefix: prefix,
		Suffix: suffix,
	})

	return nil
}
