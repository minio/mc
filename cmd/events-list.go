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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	eventsListFlags = []cli.Flag{}
)

var eventsListCmd = cli.Command{
	Name:   "list",
	Usage:  "List bucket notifications.",
	Action: mainEventsList,
	Before: setGlobalsFromContext,
	Flags:  append(eventsListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ARN [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. List notification configurations associated to a specific arn
     $ {{.HelpName}} myminio/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue

   2. List all notification configurations
     $ {{.HelpName}} s3/mybucket

`,
}

// checkEventsListSyntax - validate all the passed arguments
func checkEventsListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 && len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// eventsListMessage container
type eventsListMessage struct {
	Status string   `json:"status"`
	ID     string   `json:"id"`
	Events []string `json:"events"`
	Prefix string   `json:"prefix"`
	Suffix string   `json:"suffix"`
	Arn    string   `json:"arn"`
}

func (u eventsListMessage) JSON() string {
	u.Status = "success"
	eventsListMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(eventsListMessageJSONBytes)
}

func (u eventsListMessage) String() string {
	msg := console.Colorize("ARN", fmt.Sprintf("%s   ", u.Arn))
	for i, event := range u.Events {
		msg += console.Colorize("Events", event)
		if i != len(u.Events)-1 {
			msg += ","
		}
	}
	msg += console.Colorize("Filter", fmt.Sprintf("   Filter: "))
	if u.Prefix != "" {
		msg += console.Colorize("Filter", fmt.Sprintf("prefix=\"%s\"", u.Prefix))
	}
	if u.Suffix != "" {
		msg += console.Colorize("Filter", fmt.Sprintf("suffix=\"%s\"", u.Suffix))
	}
	return msg
}

func mainEventsList(ctx *cli.Context) error {
	console.SetColor("ARN", color.New(color.FgGreen, color.Bold))
	console.SetColor("Events", color.New(color.FgCyan, color.Bold))
	console.SetColor("Filter", color.New(color.Bold))

	checkEventsListSyntax(ctx)

	args := ctx.Args()
	path := args[0]
	arn := ""
	if len(args) > 1 {
		arn = args[1]
	}

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	s3Client, ok := client.(*s3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	configs, err := s3Client.ListNotificationConfigs(arn)
	fatalIf(err, "Cannot list notifications on the specified bucket.")

	for _, config := range configs {
		printMsg(eventsListMessage{Events: config.Events,
			Prefix: config.Prefix,
			Suffix: config.Suffix,
			Arn:    config.Arn,
			ID:     config.ID})
	}

	return nil
}
