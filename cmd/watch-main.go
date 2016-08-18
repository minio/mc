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
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	watchFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of watch.",
		},
		cli.StringFlag{
			Name:  "events",
			Value: "put,delete",
			Usage: "Filter specific type of events. Defaults to all events by default.",
		},
		cli.StringFlag{
			Name:  "prefix",
			Usage: "Filter events for a prefix",
		},
		cli.StringFlag{
			Name:  "suffix",
			Usage: "Filter events for a suffix",
		},
	}
)

var watchCmd = cli.Command{
	Name:   "watch",
	Usage:  "Watch for events on object storage and filesystem.",
	Action: mainWatch,
	Flags:  append(watchFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Watch new S3 operations on a minio server
      $ mc {{.Name}} play/testbucket

   2. Watch new events for a specific prefix "output/"  on minio server.
      $ mc {{.Name}} play/testbucket --prefix "output/"

   3. Watch new events for a specific suffix ".jpg" on minio server.
      $ mc {{.Name}} play/testbucket --suffix ".jpg"

   4. Watch new events on a specific prefix and suffix on minio server.
      $ mc {{.Name}} play/testbucket --suffix ".jpg" --prefix "photos/"

   5. Watch for events on local directory.
      $ mc {{.Name}} /usr/share
`,
}

// checkWatchSyntax - validate all the passed arguments
func checkWatchSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "watch", 1) // last argument is exit code
	}
}

// watchMessage container to hold one event notification
type watchMessage struct {
	Status string `json:"status"`
	Event  Event  `json:"events"`
}

func (u watchMessage) JSON() string {
	u.Status = "success"
	watchMessageJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(watchMessageJSONBytes)
}

func (u watchMessage) String() string {
	msg := console.Colorize("Time", fmt.Sprintf("[%s] ", u.Event.Time))
	if u.Event.Type == EventCreate {
		msg += console.Colorize("Size", fmt.Sprintf("%6s ", humanize.IBytes(uint64(u.Event.Size))))
	} else {
		msg += fmt.Sprintf("%6s ", "")
	}
	msg += console.Colorize("EventType", fmt.Sprintf("%s ", u.Event.Type))
	msg += console.Colorize("ObjectName", fmt.Sprintf("%s", u.Event.Path))
	return msg
}

func mainWatch(ctx *cli.Context) {
	console.SetColor("Time", color.New(color.FgGreen))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("EventType", color.New(color.FgCyan, color.Bold))
	console.SetColor("ObjectName", color.New(color.Bold))

	setGlobalsFromContext(ctx)
	checkWatchSyntax(ctx)

	args := ctx.Args()
	path := args[0]

	recursive := ctx.Bool("recursive")
	prefix := ctx.String("prefix")
	suffix := ctx.String("suffix")
	events := strings.Split(ctx.String("events"), ",")

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
				msg := watchMessage{Event: event}
				printMsg(msg)
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}
				fatalIf(err, "Cannot watch on events.")
				return
			}
		}
	}()

	wg.Wait()
}
