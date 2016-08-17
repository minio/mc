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
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	listenFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "events",
			Value: "all",
			Usage: "Filter specific type of events. Could be `put` or `delete` or `all`. Defaults is all",
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

var listenCmd = cli.Command{
	Name:   "listen",
	Usage:  "Print realtime bucket notification.",
	Action: mainListen,
	Flags:  append(listenFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Watch new S3 operations on a minio server
      $ mc {{.Name}} myminio/testbucket

   2. Watch new events on a specific parameters
      $ mc {{.Name}} myminio/testbucket --prefix "output/"
`,
}

// checkListenSyntax - validate all the passed arguments
func checkListenSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "listen", 1) // last argument is exit code
	}
}

// listenMessage container to hold one event notification
type listenMessage struct {
	Status string `json:"status"`
	Event  Event  `json:"events"`
}

func (u listenMessage) JSON() string {
	u.Status = "success"
	listenMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(listenMessageJSONBytes)
}

func (u listenMessage) String() string {
	msg := console.Colorize("Time", u.Event.Time.String()+"\t")
	msg += console.Colorize("EventType", u.Event.Type+"\t")
	msg += console.Colorize("ObjectName", u.Event.Path)
	return msg
}

func mainListen(ctx *cli.Context) {
	console.SetColor("Time", color.New(color.FgYellow, color.Bold))
	console.SetColor("EventType", color.New(color.FgCyan, color.Bold))
	console.SetColor("EventName", color.New(color.Bold))

	setGlobalsFromContext(ctx)
	checkListenSyntax(ctx)

	args := ctx.Args()
	path := args[0]

	recursive := ctx.Bool("recursive")
	events := ctx.String("events")
	prefix := ctx.String("prefix")
	suffix := ctx.String("suffix")

	s3Client, pErr := newClient(path)
	if pErr != nil {
		fatalIf(pErr.Trace(), "Cannot parse the provided url.")
	}

	params := watchParams{recursive: recursive,
		accountID: fmt.Sprintf("%d", time.Now().Unix()),
		events:    events,
		prefix:    prefix,
		suffix:    suffix}

	// Start watching on events
	wo, err := s3Client.Watch(params)
	fatalIf(err, "Cannot watch on the specified bucket.")

	trapCh := signalTrap(os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-trapCh:
				s3Client.Unwatch(params)
				return
			case event, ok := <-wo.Events():
				if !ok {
					return
				}
				msg := listenMessage{Event: event}
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
