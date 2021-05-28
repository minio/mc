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
	"context"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var (
	eventAddFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "event",
			Value: "put,delete,get",
			Usage: "filter specific type of event. Defaults to all event",
		},
		cli.StringFlag{
			Name:  "prefix",
			Usage: "filter event associated to the specified prefix",
		},
		cli.StringFlag{
			Name:  "suffix",
			Usage: "filter event associated to the specified suffix",
		},
		cli.BoolFlag{
			Name:  "ignore-existing, p",
			Usage: "ignore if event already exists",
		},
	}
)

var eventAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a new bucket notification",
	Action:       mainEventAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(eventAddFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ARN [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable bucket notification with a specific ARN
    {{.Prompt}} {{.HelpName}} myminio/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue

  2. Enable bucket notification with filters parameters
    {{.Prompt}} {{.HelpName}} s3/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue --event put,delete,get --prefix photos/ --suffix .jpg

  3. Ignore duplicate bucket notification with -p flag
    {{.Prompt}} {{.HelpName}} s3/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue -p --event put,delete,get --prefix photos/ --suffix .jpg

  4. Enable bucket notification for Replication and ILM transition events to a specific ARN
    {{.Prompt}} {{.HelpName}} myminio/mysourcebucket arn:aws:sqs:us-west-2:444455556666:your-queue --event replica,ilm
`,
}

// checkEventAddSyntax - validate all the passed arguments
func checkEventAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "add", 1) // last argument is exit code
	}
}

// eventAddMessage container
type eventAddMessage struct {
	ARN    string   `json:"arn"`
	Event  []string `json:"event"`
	Prefix string   `json:"prefix"`
	Suffix string   `json:"suffix"`
	Status string   `json:"status"`
}

// JSON jsonified update message.
func (u eventAddMessage) JSON() string {
	u.Status = "success"
	eventAddMessageJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(eventAddMessageJSONBytes)
}

func (u eventAddMessage) String() string {
	msg := console.Colorize("Event", "Successfully added "+u.ARN)
	return msg
}

func mainEventAdd(cliCtx *cli.Context) error {
	ctx, cancelEventAdd := context.WithCancel(globalContext)
	defer cancelEventAdd()

	console.SetColor("Event", color.New(color.FgGreen, color.Bold))

	checkEventAddSyntax(cliCtx)

	args := cliCtx.Args()
	path := args[0]
	arn := args[1]
	ignoreExisting := cliCtx.Bool("p")

	event := strings.Split(cliCtx.String("event"), ",")
	prefix := cliCtx.String("prefix")
	suffix := cliCtx.String("suffix")

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Unable to parse the provided url.")
	}

	s3Client, ok := client.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	err = s3Client.AddNotificationConfig(ctx, arn, event, prefix, suffix, ignoreExisting)
	fatalIf(err, "Unable to enable notification on the specified bucket.")
	printMsg(eventAddMessage{
		ARN:    arn,
		Event:  event,
		Prefix: prefix,
		Suffix: suffix,
	})

	return nil
}
