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
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	eventListFlags = []cli.Flag{}
)

var eventListCmd = cli.Command{
	Name:   "list",
	Usage:  "list bucket notifications",
	Action: mainEventList,
	Before: setGlobalsFromContext,
	Flags:  append(eventListFlags, globalFlags...),
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
