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
	"sync"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	notifyListenFlags = []cli.Flag{
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
		cli.BoolFlag{
			Name:  "recursive",
			Usage: "Indicate if we should watch events in sub-directories.",
		},
	}
)

var notifyListenCmd = cli.Command{
	Name:   "listen",
	Usage:  "Print realtime bucket notification.",
	Action: mainNotifyListen,
	Flags:  append(notifyListenFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc notify {{.Name}} - {{.Usage}}

USAGE:
   mc notify {{.Name}} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Watch new S3 operations on a minio server
      $ mc notify {{.Name}} myminio/testbucket

   2. Watch new events on a specific region and account id
      $ mc notify {{.Name}} myminio/testbucket --account-region us-west-2 --account-id 81132344
`,
}

// checkNotifyListenSyntax - validate all the passed arguments
func checkNotifyListenSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "listen", 1) // last argument is exit code
	}
}

// notifyListenMessage container to hold one event notification
type notifyListenMessage struct {
	Status string `json:"status"`
	Event  Event  `json:"event"`
}

// JSON jsonified update message.
func (u notifyListenMessage) JSON() string {
	u.Status = "success"
	notifyMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(notifyMessageJSONBytes)
}

func (u notifyListenMessage) String() string {
	msg := console.Colorize("Time", u.Event.Time.String()+"\t")
	msg += console.Colorize("EventType", u.Event.Type+"\t")
	msg += console.Colorize("ObjectName", u.Event.Path)
	return msg
}

func mainNotifyListen(ctx *cli.Context) {

	console.SetColor("Time", color.New(color.FgYellow, color.Bold))
	console.SetColor("EventType", color.New(color.FgCyan, color.Bold))
	console.SetColor("EventName", color.New(color.Bold))

	setGlobalsFromContext(ctx)

	checkNotifyListenSyntax(ctx)

	args := ctx.Args()
	path := args[0]

	region := ctx.String("account-region")
	accountID := ctx.String("account-id")
	recursive := ctx.Bool("recursive")

	s3Client, pErr := newClient(path)
	if pErr != nil {
		fatalIf(pErr.Trace(), "Cannot parse the provided url.")
	}

	// Start watching on events
	wo, err := s3Client.Watch(watchParams{recursive: recursive, accountRegion: region, accountID: accountID})

	fatalIf(err, "Cannot watch on the specified bucket.")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event, ok := <-wo.Events():
				if !ok {
					return
				}
				msg := notifyListenMessage{Event: event}
				printMsg(msg)
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}
				fatalIf(err, "Cannot listen to events.")
				return
			}
		}
	}()

	wg.Wait()
}
