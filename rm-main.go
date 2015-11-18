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

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// rm specific flags.
var (
	rmFlagRecursive = cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "Remove recursively.",
	}
	rmFlagForce = cli.BoolFlag{
		Name:  "force",
		Usage: "Force a dangerous remove operation.",
	}
	rmFlagIncomplete = cli.BoolFlag{
		Name:  "incomplete, I",
		Usage: "Remove an incomplete upload(s).",
	}
	rmFlagFake = cli.BoolFlag{
		Name:  "fake",
		Usage: "Perform a fake remove operation.",
	}
	rmFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of rm.",
	}
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:   "rm",
	Usage:  "Remove file or bucket [WARNING: Use with care].",
	Action: mainRm,
	Flags:  []cli.Flag{rmFlagRecursive, rmFlagForce, rmFlagIncomplete, rmFlagFake, rmFlagHelp},
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

   2. Remove contents of a folder, excluding its sub-folders.
     $ mc {{.Name}} --force https://s3.amazonaws.com/jazz-songs/louis/

   3. Remove contents of a folder recursively.
     $ mc {{.Name}} --force --recursive https://s3.amazonaws.com/jazz-songs/louis/

   4. Remove all matching objects with this prefix.
     $ mc {{.Name}} --force --force https://s3.amazonaws.com/ogg/gunmetal

   5. Drop an incomplete upload of an object.
      $ mc {{.Name}} --incomplete https://s3.amazonaws.com/jazz-songs/louis/file01.mp3

   6. Drop all incomplete uploads recursively matching this prefix.
      $ mc {{.Name}} --incomplete --force --recursive https://s3.amazonaws.com/jazz-songs/louis/
`,
}

// Structured message depending on the type of console.
type rmMessage struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

// Colorized message for console printing.
func (r rmMessage) String() string {
	return console.Colorize("Remove", fmt.Sprintf("Removed ‘%s’.", r.URL))
}

// JSON'ified message for scripting.
func (r rmMessage) JSON() string {
	msgBytes, e := json.Marshal(r)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate command line arguments.
func checkRmSyntax(ctx *cli.Context) {
	isForce := ctx.Bool("force")
	isRecursive := ctx.Bool("recursive")

	if !ctx.Args().Present() {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
	}

	if isRecursive && !isForce {
		fatalIf(errDummy().Trace(),
			"Recursive removal requires --force option. Please review carefully before performing this *DANGEROUS* operation.")
	}
}

// Remove a single object.
func rm(url string, isIncomplete, isFake bool) *probe.Error {
	clnt, err := url2Client(url)
	if err != nil {
		return err.Trace(url)
	}

	if isFake { // It is a fake remove. Return success.
		return nil
	}

	if err = clnt.Remove(isIncomplete); err != nil {
		return err.Trace(url)
	}

	return nil
}

// Remove all objects recursively.
func rmAll(url string, isRecursive, isIncomplete, isFake bool) {
	// Initialize new client.
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(url), "Invalid URL ‘"+url+"’.")
		return // End of journey.
	}

	/* Disable recursion and only list this folder's contents. We
	perform manual depth-first recursion ourself here. */
	nonRecursive := false
	for entry := range clnt.List(nonRecursive, isIncomplete) {
		if entry.Err != nil {
			errorIf(entry.Err.Trace(url), "Unable to list ‘"+url+"’.")
			return // End of journey.
		}

		if entry.Type.IsDir() && isRecursive {
			// Add separator at the end to remove all its contents.
			url := entry.URL
			url.Path = strings.TrimSuffix(entry.URL.Path, string(entry.URL.Separator)) + string(entry.URL.Separator)

			// Recursively remove contents of this directory.
			rmAll(url.String(), isRecursive, isIncomplete, isFake)
		}

		// Regular type.
		if err = rm(entry.URL.String(), isIncomplete, isFake); err != nil {
			errorIf(err.Trace(entry.URL.String()), "Unable to remove ‘"+entry.URL.String()+"’.")
			continue
		}
		printMsg(rmMessage{Status: "success", URL: entry.URL.String()})
	}
}

// main for rm command.
func mainRm(ctx *cli.Context) {
	checkRmSyntax(ctx)

	// rm specific flags.
	isForce := ctx.Bool("force")
	isIncomplete := ctx.Bool("incomplete")
	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	// Parse args.
	URLs, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	// Support multiple targets.
	for _, url := range URLs {
		if isRecursive && isForce {
			rmAll(url, isRecursive, isIncomplete, isFake)
		} else {
			if err := rm(url, isIncomplete, isFake); err != nil {
				errorIf(err.Trace(url), "Unable to remove ‘"+url+"’.")
				continue
			}
			printMsg(rmMessage{Status: "success", URL: url})
		}
	}
}
