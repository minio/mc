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
	"errors"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var (
	eventRemoveFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "force removing all bucket notifications",
		},
		cli.StringFlag{
			Name:  "event",
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
	}
)

var eventRemoveCmd = cli.Command{
	Name:         "remove",
	Usage:        "remove a bucket notification; '--force' removes all bucket notifications",
	Action:       mainEventRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(eventRemoveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [ARN] [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove bucket notification associated to a specific arn
    {{.Prompt}} {{.HelpName}} myminio/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue

  2. Remove all bucket notifications. --force flag is mandatory here
    {{.Prompt}} {{.HelpName}} myminio/mybucket --force
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

func mainEventRemove(cliCtx *cli.Context) error {
	ctx, cancelEventRemove := context.WithCancel(globalContext)
	defer cancelEventRemove()

	console.SetColor("Event", color.New(color.FgGreen, color.Bold))

	checkEventRemoveSyntax(cliCtx)

	args := cliCtx.Args()
	path := args.Get(0)

	arn := ""
	if len(args) == 2 {
		arn = args.Get(1)
	}

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Unable to parse the provided url.")
	}

	s3Client, ok := client.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	// flags for the attributes of the even
	event := cliCtx.String("event")
	prefix := cliCtx.String("prefix")
	suffix := cliCtx.String("suffix")

	err = s3Client.RemoveNotificationConfig(ctx, arn, event, prefix, suffix)
	if err != nil {
		fatalIf(err, "Unable to disable notification on the specified bucket.")
	}

	printMsg(eventRemoveMessage{ARN: arn})

	return nil
}
