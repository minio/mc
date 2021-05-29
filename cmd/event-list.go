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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var (
	eventListFlags = []cli.Flag{}
)

var eventListCmd = cli.Command{
	Name:         "list",
	Usage:        "list bucket notifications",
	Action:       mainEventList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(eventListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ARN [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List notification configurations associated to a specific arn
    {{.Prompt}} {{.HelpName}} myminio/mybucket arn:aws:sqs:us-west-2:444455556666:your-queue

  2. List all notification configurations
    {{.Prompt}} {{.HelpName}} s3/mybucket
`,
}

// checkEventListSyntax - validate all the passed arguments
func checkEventListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 && len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

// eventListMessage container
type eventListMessage struct {
	Status string   `json:"status"`
	ID     string   `json:"id"`
	Event  []string `json:"event"`
	Prefix string   `json:"prefix"`
	Suffix string   `json:"suffix"`
	Arn    string   `json:"arn"`
}

func (u eventListMessage) JSON() string {
	u.Status = "success"
	eventListMessageJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(eventListMessageJSONBytes)
}

func (u eventListMessage) String() string {
	msg := console.Colorize("ARN", fmt.Sprintf("%s   ", u.Arn))
	for i, event := range u.Event {
		msg += console.Colorize("Event", event)
		if i != len(u.Event)-1 {
			msg += ","
		}
	}
	msg += console.Colorize("Filter", "   Filter: ")
	if u.Prefix != "" {
		msg += console.Colorize("Filter", fmt.Sprintf("prefix=\"%s\"", u.Prefix))
	}
	if u.Suffix != "" {
		msg += console.Colorize("Filter", fmt.Sprintf("suffix=\"%s\"", u.Suffix))
	}
	return msg
}

func mainEventList(cliCtx *cli.Context) error {
	ctx, cancelEventList := context.WithCancel(globalContext)
	defer cancelEventList()

	console.SetColor("ARN", color.New(color.FgGreen, color.Bold))
	console.SetColor("Event", color.New(color.FgCyan, color.Bold))
	console.SetColor("Filter", color.New(color.Bold))

	checkEventListSyntax(cliCtx)

	args := cliCtx.Args()
	path := args[0]
	arn := ""
	if len(args) > 1 {
		arn = args[1]
	}

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Unable to parse the provided url.")
	}

	s3Client, ok := client.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	configs, err := s3Client.ListNotificationConfigs(ctx, arn)
	fatalIf(err, "Unable to list notifications on the specified bucket.")

	for _, config := range configs {
		printMsg(eventListMessage{
			Event:  config.Events,
			Prefix: config.Prefix,
			Suffix: config.Suffix,
			Arn:    config.Arn,
			ID:     config.ID})
	}

	return nil
}
