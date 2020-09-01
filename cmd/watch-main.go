/*
 * MinIO Client (C) 2016-2019 MinIO, Inc.
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
	"strings"
	"sync"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	watchFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "events",
			Value: "put,delete,get",
			Usage: "filter specific types of events; defaults to all events by default",
		},
		cli.StringFlag{
			Name:  "prefix",
			Usage: "filter events for a prefix",
		},
		cli.StringFlag{
			Name:  "suffix",
			Usage: "filter events for a suffix",
		},
		cli.BoolFlag{
			Name:  "recursive",
			Usage: "recursively watch for events",
		},
	}
)

var watchCmd = cli.Command{
	Name:   "watch",
	Usage:  "listen for object notification events",
	Action: mainWatch,
	Before: setGlobalsFromContext,
	Flags:  append(watchFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] PATH
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
  1. Watch new S3 operations on a MinIO server
     {{.Prompt}} {{.HelpName}} play/testbucket

  2. Watch new events for a specific prefix "output/"  on MinIO server.
     {{.Prompt}} {{.HelpName}} --prefix "output/" play/testbucket

  3. Watch new events for a specific suffix ".jpg" on MinIO server.
     {{.Prompt}} {{.HelpName}} --suffix ".jpg" play/testbucket

  4. Watch new events on a specific prefix and suffix on MinIO server.
     {{.Prompt}} {{.HelpName}} --suffix ".jpg" --prefix "photos/" play/testbucket

  5. Site level watch (except new buckets created after running this command)
     {{.Prompt}} {{.HelpName}} play/

  6. Watch for events on local directory.
     {{.Prompt}} {{.HelpName}} /usr/share
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
	Event  struct {
		Time string    `json:"time"`
		Size int64     `json:"size"`
		Path string    `json:"path"`
		Type EventType `json:"type"`
	} `json:"events"`
	Source struct {
		Host      string `json:"host,omitempty"`
		Port      string `json:"port,omitempty"`
		UserAgent string `json:"userAgent,omitempty"`
	} `json:"source,omitempty"`
}

func (u watchMessage) JSON() string {
	u.Status = "success"
	watchMessageJSONBytes, e := json.MarshalIndent(u, "", " ")
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
	msg += console.Colorize("ObjectName", u.Event.Path)
	return msg
}

func mainWatch(cliCtx *cli.Context) error {
	console.SetColor("Time", color.New(color.FgGreen))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("EventType", color.New(color.FgCyan, color.Bold))
	console.SetColor("ObjectName", color.New(color.Bold))

	checkWatchSyntax(cliCtx)

	args := cliCtx.Args()
	path := args[0]

	prefix := cliCtx.String("prefix")
	suffix := cliCtx.String("suffix")
	events := strings.Split(cliCtx.String("events"), ",")
	recursive := cliCtx.Bool("recursive")

	s3Client, pErr := newClient(path)
	if pErr != nil {
		fatalIf(pErr.Trace(), "Unable to parse the provided url.")
	}

	options := WatchOptions{
		Recursive: recursive,
		Events:    events,
		Prefix:    prefix,
		Suffix:    suffix,
	}

	ctx, cancelWatch := context.WithCancel(globalContext)
	defer cancelWatch()

	// Start watching on events
	wo, err := s3Client.Watch(ctx, options)
	fatalIf(err, "Unable to watch on the specified bucket.")

	// Initialize.. waitgroup to track the go-routine.
	var wg sync.WaitGroup

	// Increment wait group to wait subsequent routine.
	wg.Add(1)

	// Start routine to watching on events.
	go func() {
		defer wg.Done()

		// Wait for all events.
		for {
			select {
			case <-globalContext.Done():
				// Signal received we are done.
				close(wo.DoneChan)
				return
			case events, ok := <-wo.Events():
				if !ok {
					return
				}
				for _, event := range events {
					msg := watchMessage{}
					msg.Event.Path = event.Path
					msg.Event.Size = event.Size
					msg.Event.Time = event.Time
					msg.Event.Type = event.Type
					msg.Source.Host = event.Host
					msg.Source.Port = event.Port
					msg.Source.UserAgent = event.UserAgent
					printMsg(msg)
				}
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}
				if err != nil {
					errorIf(err, "Unable to watch for events.")
					return
				}
			}
		}
	}()

	// Wait on the routine to be finished or exit.
	wg.Wait()

	return nil
}
