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
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// rm specific flags.
var (
	rmFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of rm.",
		},
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
			Usage: "Remove an incomplete upload(s).",
		},
		cli.BoolFlag{
			Name:  "fake",
			Usage: "Perform a fake remove operation.",
		},
	}
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:   "rm",
	Usage:  "Remove file or bucket [WARNING: Use with care].",
	Action: mainRm,
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

   2. Remove contents of a folder, excluding its sub-folders.
     $ mc {{.Name}} --force s3/jazz-songs/louis/

   3. Remove contents of a folder recursively.
     $ mc {{.Name}} --force --recursive s3/jazz-songs/louis/

   4. Remove all matching objects with this prefix.
     $ mc {{.Name}} --force s3/ogg/gunmetal

   5. Drop an incomplete upload of an object.
      $ mc {{.Name}} --incomplete s3/jazz-songs/louis/file01.mp3

   6. Drop all incomplete uploads recursively matching this prefix.
      $ mc {{.Name}} --incomplete --force --recursive s3/jazz-songs/louis/
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
	// Set command flags from context.
	isForce := ctx.Bool("force")
	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")

	if !ctx.Args().Present() {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
	}

	// For all recursive operations make sure to check for 'force' flag.
	if isRecursive && !isForce && !isFake {
		fatalIf(errDummy().Trace(),
			"Recursive removal requires --force option. Please review carefully before performing this *DANGEROUS* operation.")
	}
}

// Remove a single object.
func rm(targetAlias, targetURL string, isIncomplete, isFake bool) *probe.Error {
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	if isFake { // It is a fake remove. Return success.
		return nil
	}

	if err = clnt.Remove(isIncomplete); err != nil {
		return err.Trace(targetURL)
	}

	return nil
}

// Remove all objects recursively.
func rmAll(targetAlias, targetURL string, isRecursive, isIncomplete, isFake bool) {
	// Initialize new client.
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		errorIf(err.Trace(targetURL), "Invalid URL ‘"+targetURL+"’.")
		return // End of journey.
	}

	/* Disable recursion and only list this folder's contents. We
	perform manual depth-first recursion ourself here. */
	nonRecursive := false
	for entry := range clnt.List(nonRecursive, isIncomplete) {
		if entry.Err != nil {
			errorIf(entry.Err.Trace(targetURL), "Unable to list ‘"+targetURL+"’.")
			return // End of journey.
		}

		if entry.Type.IsDir() && isRecursive {
			// Add separator at the end to remove all its contents.
			url := entry.URL
			url.Path = strings.TrimSuffix(entry.URL.Path, string(entry.URL.Separator)) + string(entry.URL.Separator)

			// Recursively remove contents of this directory.
			rmAll(targetAlias, url.String(), isRecursive, isIncomplete, isFake)
		}

		// Regular type.
		if err = rm(targetAlias, entry.URL.String(), isIncomplete, isFake); err != nil {
			errorIf(err.Trace(entry.URL.String()), "Unable to remove ‘"+entry.URL.String()+"’.")
			continue
		}
		// Construct user facing message and path.
		entryPath := filepath.ToSlash(filepath.Join(targetAlias, entry.URL.Path))
		printMsg(rmMessage{Status: "success", URL: entryPath})
	}
}

// main for rm command.
func mainRm(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'rm' cli arguments.
	checkRmSyntax(ctx)

	// rm specific flags.
	isForce := ctx.Bool("force")
	isIncomplete := ctx.Bool("incomplete")
	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	// Support multiple targets.
	for _, url := range ctx.Args() {
		targetAlias, targetURL, _ := mustExpandAlias(url)
		if isRecursive && isForce || isFake {
			rmAll(targetAlias, targetURL, isRecursive, isIncomplete, isFake)
		} else {
			if err := rm(targetAlias, targetURL, isIncomplete, isFake); err != nil {
				errorIf(err.Trace(url), "Unable to remove ‘"+url+"’.")
				continue
			}
			printMsg(rmMessage{Status: "success", URL: url})
		}
	}
}
