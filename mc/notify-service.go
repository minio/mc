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

package mc

import (
	"encoding/json"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	notifyServiceFlags = []cli.Flag{
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

var notifyServiceCmd = cli.Command{
	Name:   "service",
	Usage:  "Enable/Disable lambda notification",
	Action: mainNotifyService,
	Flags:  append(notifyServiceFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc notify {{.Name}} - {{.Usage}}

USAGE:
   mc notify {{.Name}} {enable|disable} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Enable bucket notification with a specific region and account id
      $ mc notify {{.Name}} enable myminio/testbucket
   2. Disable bucket notification with specific region and accnout id
      $ mc notify {{.Name}} disable myminio/testbucket --account-region us-west-2 --account-id 81132344

`,
}

// checkNotifyEnableSyntax - validate all the passed arguments
func checkNotifyServiceSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "service", 1) // last argument is exit code
	}
	if ctx.Args()[0] != "enable" && ctx.Args()[0] != "disable" {
		cli.ShowCommandHelpAndExit(ctx, "service", 1)
	}
}

// notifyServiceMessage container
type notifyServiceMessage struct {
	Status string `json:"status"`
}

// JSON jsonified update message.
func (u notifyServiceMessage) JSON() string {
	u.Status = "success"
	notifyMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(notifyMessageJSONBytes)
}

func (u notifyServiceMessage) String() string {
	msg := console.Colorize("Service", "Successfully accomplished.")
	return msg
}

func mainNotifyService(ctx *cli.Context) {
	console.SetColor("Service", color.New(color.FgGreen, color.Bold))

	setGlobalsFromContext(ctx)
	checkNotifyServiceSyntax(ctx)

	args := ctx.Args()
	serviceStatus := args[0]
	path := args[1]

	region := ctx.String("account-region")
	accountID := ctx.String("account-id")

	client, err := newClient(path)
	if err != nil {
		fatalIf(err.Trace(), "Cannot parse the provided url.")
	}

	s3Client, ok := client.(*s3Client)
	if !ok {
		fatalIf(errDummy().Trace(), "The provided url doesn't point to a S3 server.")
	}

	err = s3Client.ToogleLambdaNotificationStatus(serviceStatus == "enable", region, accountID)
	fatalIf(err, "Cannot enable notification on the specified bucket.")
	printMsg(notifyServiceMessage{})
}
