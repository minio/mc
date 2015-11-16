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
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// rm specific flags.
var (
	rmFlagForce = cli.BoolFlag{
		Name:  "force",
		Usage: "Force a dangerous remove operation.",
	}

	rmFlagIncomplete = cli.BoolFlag{
		Name:  "incomplete, I",
		Usage: "Remove an incomplete upload(s).",
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
	Flags:  []cli.Flag{rmFlagForce, rmFlagIncomplete, rmFlagHelp},
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Remove an object.
     $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/file01.mp3

   2. Remove a file.
      $ mc {{.Name}} 1999/old-backup.tgz

   3. Remove contents of a folder recursively.
     $ mc {{.Name}} --force https://s3.amazonaws.com/jazz-songs/louis/...

   4  Remove contents of a folder recursively and folder itself.
      $ mc {{.Name}} old/photos...

   5. Remove a bucket and all its contents recursively.
     $ mc {{.Name}} --force https://s3.amazonaws.com/jazz-songs...

   6. Remove all matching objects with this prefix.
     $ mc {{.Name}} --force https://s3.amazonaws.com/ogg/gunmetal...

   7. Cancel an incomplete upload of an object.
      $ mc {{.Name}} --incomplete https://s3.amazonaws.com/jazz-songs/louis/file01.mp3

   8. Remove all incomplete uploads recursively matching this prefix.
      $ mc {{.Name}} --incomplete --force https://s3.amazonaws.com/jazz-songs/louis/...
`,
}

// Structured message depending on the type of console.
type rmMessage struct {
	URL string `json:"url"`
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
	args := ctx.Args()
	isForce := ctx.Bool("force")

	if !args.Present() {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
	}

	URLs, err := args2URLs(args)
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	// If input validation fails then provide context sensitive help without displaying generic help message.
	// The context sensitive help is shown per argument instead of all arguments to keep the help display
	// as well as the code simple. Also most of the times there will be just one arg
	for _, url := range URLs {
		u := client.NewURL(url)
		if strings.HasSuffix(url, string(u.Separator)) {
			fatalIf(errDummy().Trace(),
				"‘"+url+"’ is a folder. To remove this folder recursively, please try ‘"+url+"...’ as argument.")
		}
		if isURLRecursive(url) && !isForce {
			fatalIf(errDummy().Trace(),
				"Recursive removal requires --force option. Please review carefully before performing this operation.")
		}
	}
}

// Remove a single object.
func rm(url string, isIncomplete bool) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(url), "Invalid URL ‘"+url+"’.")
		return
	}

	if err = clnt.Remove(isIncomplete); err != nil {
		fatalIf(err.Trace(url), "Unable to remove ‘"+url+"’.")
	}

	printMsg(rmMessage{url})
}

// Remove all objects recursively.
func rmAll(url string, isIncomplete bool) {
	// Initialize new client.
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(url), "Invalid URL ‘"+url+"’.")
		return // End of journey.
	}
	isRecursive := false // Disable recursion and only list this folder's contents.
	for entry := range clnt.List(isRecursive, isIncomplete) {
		if entry.Err != nil {
			errorIf(entry.Err.Trace(url), "Unable to list ‘"+url+"’.")
			return // End of journey.
		}

		if entry.Content.Type.IsDir() {
			// Add separator at the end to remove all its contents.
			url := entry.Content.URL.String()
			u := client.NewURL(url)
			url = url + string(u.Separator)

			// Recursively remove contents of this directory.
			rmAll(url, isIncomplete)
		}
		// Regular type.
		rm(entry.Content.URL.String(), isIncomplete)
	}
}

// main for rm command.
func mainRm(ctx *cli.Context) {
	checkRmSyntax(ctx)

	// rm specific flags.
	isForce := ctx.Bool("force")
	isIncomplete := ctx.Bool("incomplete")

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	// Parse args.
	URLs, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	// Support multiple targets.
	for _, url := range URLs {
		if isURLRecursive(url) && isForce {
			url := stripRecursiveURL(url)
			removeTopFolder := false
			// find if the URL is dir or not.
			_, content, err := url2Stat(url)
			if err != nil && !prefixExists(url) {
				fatalIf(err.Trace(url), "Unable to stat ‘"+url+"’.")
			}

			if err == nil && content.Type.IsDir() {
				/* Determine whether to remove the top folder or only its
				contents. If the URL does not end with a separator, then
				include the top folder as well, otherwise not. */
				u := client.NewURL(url)
				if !strings.HasSuffix(url, string(u.Separator)) {
					// Add separator at the end to remove all its contents.
					url = url + string(u.Separator)
					// Remember to remove the top most folder.
					removeTopFolder = true
				}
			}
			// Remove contents of this folder.
			rmAll(url, isIncomplete)
			if removeTopFolder {
				// Remove top folder as well.
				rm(url, isIncomplete)
			}

		} else {
			rm(url, isIncomplete)
		}
	}
}
