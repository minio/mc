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
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// default older time.
const defaultOlderTime = time.Hour

// rm specific flags.
var (
	rmFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Show this help.",
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
			Name:  "prefix",
			Usage: "Remove objects matching this prefix.",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "Remove an incomplete upload(s).",
		},
		cli.BoolFlag{
			Name:  "fake",
			Usage: "Perform a fake remove operation.",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "Read object list from STDIN.",
		},
		cli.StringFlag{
			Name:  "older",
			Usage: "Remove object only if its created older than given time.",
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

   2. Remove all objects from music/ogg folder matching this prefix, excluding its sub-folders.
      $ mc {{.Name}} --prefix s3/music/ogg/gunmetal

   3. Remove contents of a folder, excluding its sub-folders.
      $ mc {{.Name}} --prefix s3/jazz-songs/louis/

   4. Remove contents of a folder recursively.
      $ mc {{.Name}} --recursive s3/jazz-songs/louis/

   5. Drop an incomplete upload of an object.
      $ mc {{.Name}} --incomplete s3/jazz-songs/louis/file01.mp4

   6. Remove all matching objects whose names are read from STDIN.
      $ mc {{.Name}} --force --stdin

   7. Remove object only if its created older than one day.
      $ mc {{.Name}} --force --older=24h s3/jazz-songs/louis/
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
	isPrefix := ctx.Bool("prefix")
	isRecursive := ctx.Bool("recursive")
	isStdin := ctx.Bool("stdin")
	olderString := ctx.String("older")

	if olderString != "" {
		if older, err := time.ParseDuration(olderString); err != nil {
			fatalIf(errDummy().Trace(), "Invalid older time format.")
		} else if older < defaultOlderTime {
			fatalIf(errDummy().Trace(), "older time should not be less than one hour.")
		}

		if !isStdin && !ctx.Args().Present() {
			exitCode := 1
			cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
		}
	}

	if !ctx.Args().Present() && !isStdin {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
	}

	// For all recursive operations make sure to check for 'force' flag.
	if (isPrefix || isRecursive || isStdin) && !isForce {
		fatalIf(errDummy().Trace(),
			"Removal requires --force option. This operational is irreversible. Please review carefully before performing this *DANGEROUS* operation.")
	}
}

// Remove a single object.
func rmObject(targetAlias, targetURL string, isIncomplete bool) *probe.Error {
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	if err = clnt.Remove(isIncomplete); err != nil {
		return err.Trace(targetURL)
	}

	return nil
}

// Remove a single object.
func rm(targetAlias, targetURL string, isIncomplete, isFake bool, older time.Duration) *probe.Error {
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	// Check whether object is created older than given time only if older is >= one hour.
	if older >= defaultOlderTime {
		info, err := clnt.Stat()
		if err != nil {
			return err.Trace(targetURL)
		}

		now := time.Now().UTC()
		timeDiff := now.Sub(info.Time)
		if timeDiff < older {
			// time difference of info.Time with current time is less than older duration.
			return nil
		}
	}

	if !isFake {
		if err := rmObject(targetAlias, targetURL, isIncomplete); err != nil {
			return err.Trace(targetURL)
		}
	}

	return nil
}

// Remove all objects recursively.
func rmAll(targetAlias, targetURL, prefix string, isRecursive, isIncomplete, isFake bool, older time.Duration) {
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

		if !strings.HasPrefix(path.Base(entry.URL.Path), prefix) {
			// Skip the entry if it doesn't starts with the prefix.
			continue
		}

		if entry.Type.IsDir() && isRecursive {
			// Add separator at the end to remove all its contents.
			url := entry.URL
			url.Path = strings.TrimSuffix(entry.URL.Path, string(entry.URL.Separator)) + string(entry.URL.Separator)

			// Recursively remove contents of this directory.
			rmAll(targetAlias, url.String(), prefix, isRecursive, isIncomplete, isFake, older)
		}

		// Check whether object is created older than given time only if older is >= one hour.
		if older >= defaultOlderTime {
			now := time.Now().UTC()
			timeDiff := now.Sub(entry.Time)
			if timeDiff < older {
				// time difference of info.Time with current time is less than older duration.
				continue
			}
		}

		// Regular type.
		if !isFake {
			if err = rmObject(targetAlias, entry.URL.String(), isIncomplete); err != nil {
				errorIf(err.Trace(entry.URL.String()), "Unable to remove ‘"+entry.URL.String()+"’.")
				continue
			}
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
	isPrefix := ctx.Bool("prefix")
	isIncomplete := ctx.Bool("incomplete")
	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")
	isStdin := ctx.Bool("stdin")
	olderString := ctx.String("older")
	older, _ := time.ParseDuration(olderString)

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	// Support multiple targets.
	for _, url := range ctx.Args() {
		prefix := ""
		if isPrefix {
			if !strings.HasSuffix(url, "/") {
				prefix = path.Base(url)
				url = path.Dir(url) + "/"
			} else if runtime.GOOS == "windows" && !strings.HasSuffix(url, `\`) {
				prefix = path.Base(url)
				url = path.Dir(url) + `\`
			}
		}

		targetAlias, targetURL, _ := mustExpandAlias(url)
		if (isPrefix || isRecursive) && isForce {
			rmAll(targetAlias, targetURL, prefix, isRecursive, isIncomplete, isFake, older)
		} else {
			if err := rm(targetAlias, targetURL, isIncomplete, isFake, older); err != nil {
				errorIf(err.Trace(url), "Unable to remove ‘"+url+"’.")
				continue
			}
			printMsg(rmMessage{Status: "success", URL: url})
		}
	}

	if !isStdin {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := scanner.Text()
		prefix := ""
		if isPrefix {
			if strings.HasSuffix(url, "/") {
				prefix = path.Base(url)
				url = path.Dir(url)
			} else if runtime.GOOS == "windows" && strings.HasSuffix(url, `\`) {
				prefix = path.Base(url)
				url = path.Dir(url)
			}
		}

		targetAlias, targetURL, _ := mustExpandAlias(url)
		if (isPrefix || isRecursive) && isForce {
			rmAll(targetAlias, targetURL, prefix, isRecursive, isIncomplete, isFake, older)
		} else {
			if err := rm(targetAlias, targetURL, isIncomplete, isFake, older); err != nil {
				errorIf(err.Trace(url), "Unable to remove ‘"+url+"’.")
				continue
			}
			printMsg(rmMessage{Status: "success", URL: url})
		}
	}
}
