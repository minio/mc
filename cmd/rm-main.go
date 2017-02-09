/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Day time.Duration for day.
const Day = 24 * time.Hour

// rm specific flags.
var (
	rmFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "Remove recursively.",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force a dangerous remove operation.",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "Remove incomplete uploads.",
		},
		cli.BoolFlag{
			Name:  "fake",
			Usage: "Perform a fake remove operation.",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "Read object names from STDIN.",
		},
		cli.IntFlag{
			Name:  "older-than",
			Usage: "Remove objects older than N days.",
		},
	}
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:   "rm",
	Usage:  "Remove files and objects.",
	Action: mainRm,
	Before: setGlobalsFromContext,
	Flags:  append(rmFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Remove a file.
      $ mc {{.Name}} 1999/old-backup.tgz

   2. Remove all objects recursively.
      $ mc {{.Name}} --recursive s3/jazz-songs/louis/

   3. Remove all objects older than '90' days.
      $ mc {{.Name}} --recursive --older-than=90 s3/jazz-songs/louis/

   4. Remove all objects read from STDIN.
      $ mc {{.Name}} --force --stdin

   5. Drop all incomplete uploads on 'jazz-songs' bucket.
      $ mc {{.Name}} --incomplete --recursive s3/jazz-songs/
`,
}

// Structured message depending on the type of console.
type rmMessage struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

// Colorized message for console printing.
func (r rmMessage) String() string {
	return console.Colorize("Remove", fmt.Sprintf("Removing ‘%s’.", r.Key))
}

// JSON'ified message for scripting.
func (r rmMessage) JSON() string {
	msgBytes, e := json.Marshal(r)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate command line arguments.
func checkRmSyntax(ctx *cli.Context) {
	// Set command flags from context.
	isForce := ctx.Bool("force")
	isRecursive := ctx.Bool("recursive")
	isStdin := ctx.Bool("stdin")

	if !ctx.Args().Present() && !isStdin {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
	}

	// For all recursive operations make sure to check for 'force' flag.
	if (isRecursive || isStdin) && !isForce {
		fatalIf(errDummy().Trace(),
			"Removal requires --force option. This operational is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
	}
}

func removeSingle(url string, isIncomplete bool, isFake bool, older int) error {
	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Invalid argument ‘"+url+"’.")
		return exitStatus(globalErrorExitStatus) // End of journey.
	}

	content, pErr := clnt.Stat(isIncomplete)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to remove ‘"+url+"’.")
		return exitStatus(globalErrorExitStatus)
	}
	if older > 0 {
		// Check whether object is created older than given time.
		now := time.Now().UTC()
		timeDiff := now.Sub(content.Time)
		if timeDiff < (time.Duration(older) * Day) {
			// time difference of info.Time with current time is less than older time.
			return nil
		}
	}

	printMsg(rmMessage{
		Key:  url,
		Size: content.Size,
	})

	if !isFake {
		contentCh := make(chan *clientContent, 1)
		contentCh <- &clientContent{URL: *newClientURL(targetURL)}
		close(contentCh)

		errorCh := clnt.Remove(isIncomplete, contentCh)
		for pErr := range errorCh {
			if pErr != nil {
				errorIf(pErr.Trace(url), "Failed to remove ‘"+url+"’.")
				switch pErr.ToGoError().(type) {
				case PathInsufficientPermission:
					// Ignore Permission error.
					continue
				}
				return exitStatus(globalErrorExitStatus)
			}
		}
	}
	return nil
}

func removeRecursive(url string, isIncomplete bool, isFake bool, older int) error {
	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to remove ‘"+url+"’ recursively.")
		return exitStatus(globalErrorExitStatus) // End of journey.
	}

	contentCh := make(chan *clientContent)
	errorCh := clnt.Remove(isIncomplete, contentCh)

	isEmpty := true
	isRecursive := true
	for content := range clnt.List(isRecursive, isIncomplete, DirLast) {
		isEmpty = false
		if content.Err != nil {
			errorIf(content.Err.Trace(url), "Failed to remove ‘"+url+"’ recursively.")
			switch content.Err.ToGoError().(type) {
			case PathInsufficientPermission:
				// Ignore Permission error.
				continue
			}
			close(contentCh)
			return exitStatus(globalErrorExitStatus)
		}

		if older > 0 {
			// Check whether object is created older than given time.
			now := time.Now().UTC()
			timeDiff := now.Sub(content.Time)
			if timeDiff < (time.Duration(older) * Day) {
				// time difference of info.Time with current time is less than older time.
				continue
			}
		}

		urlString := content.URL.Path
		printMsg(rmMessage{
			Key:  targetAlias + urlString,
			Size: content.Size,
		})

		if !isFake {
			sent := false
			for sent == false {
				select {
				case contentCh <- content:
					sent = true
				case pErr := <-errorCh:
					errorIf(pErr.Trace(urlString), "Failed to remove ‘"+urlString+"’.")
					switch pErr.ToGoError().(type) {
					case PathInsufficientPermission:
						// Ignore Permission error.
						continue
					}

					close(contentCh)
					return exitStatus(globalErrorExitStatus)
				}
			}
		}
	}

	close(contentCh)
	for pErr := range errorCh {
		errorIf(pErr.Trace(url), "Failed to remove ‘"+url+"’ recursively.")
		switch pErr.ToGoError().(type) {
		case PathInsufficientPermission:
			// Ignore Permission error.
			continue
		}
		return exitStatus(globalErrorExitStatus)
	}

	// As clnt.List() returns empty, we just send dummy value to behave like non-recursive.
	if isEmpty {
		printMsg(rmMessage{Key: url})
	}
	return nil
}

// main for rm command.
func mainRm(ctx *cli.Context) error {

	// check 'rm' cli arguments.
	checkRmSyntax(ctx)

	// rm specific flags.
	isIncomplete := ctx.Bool("incomplete")
	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")
	isStdin := ctx.Bool("stdin")
	older := ctx.Int("older-than")

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	var rerr error
	var err error
	// Support multiple targets.
	for _, url := range ctx.Args() {
		if isRecursive {
			err = removeRecursive(url, isIncomplete, isFake, older)
		} else {
			err = removeSingle(url, isIncomplete, isFake, older)
		}

		if rerr == nil {
			rerr = err
		}
	}

	if !isStdin {
		return rerr
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := scanner.Text()
		if isRecursive {
			err = removeRecursive(url, isIncomplete, isFake, older)
		} else {
			err = removeSingle(url, isIncomplete, isFake, older)
		}

		if rerr == nil {
			rerr = err
		}
	}

	return rerr
}
