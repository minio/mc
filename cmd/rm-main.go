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
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
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
			Usage: "Allow a recursive remove operation.",
		},
		cli.BoolFlag{
			Name:  "dangerous",
			Usage: "Allow site-wide removal of buckets and objects.",
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
			Usage: "Remove objects older than N days",
		},
		cli.IntFlag{
			Name:  "newer-than",
			Usage: "Remove objects newer than N days",
		},
		cli.StringFlag{
			Name:  "encrypt-key",
			Usage: "Encrypt object (using server-side encryption)",
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
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY: List of comma delimited prefix=secret values

EXAMPLES:
   1. Remove a file.
      $ {{.HelpName}} 1999/old-backup.tgz

   2. Remove all objects recursively from bucket 'jazz-songs' matching 'louis' prefix.
      $ {{.HelpName}} --recursive s3/jazz-songs/louis/

   3. Remove all objects older than '90' days recursively from bucket 'jazz-songs' that match 'louis' prefix.
      $ {{.HelpName}} --recursive --older-than=90 s3/jazz-songs/louis/

   4. Remove all objects newer than 7 days recursively from bucket 'pop-songs'
      $ {{.HelpName}} --recursive --newer-than=7 s3/pop-songs/

   5. Remove all objects read from STDIN.
      $ {{.HelpName}} --force --stdin

   6. Remove all buckets and objects recursively from S3 host
      $ {{.HelpName}} --recursive --dangerous s3

   7. Remove all buckets and objects older than '90' days recursively from host
      $ {{.HelpName}} --recursive --dangerous --older-than=90 s3

   8. Drop all incomplete uploads on 'jazz-songs' bucket.
      $ {{.HelpName}} --incomplete --recursive s3/jazz-songs/

   9. Remove an encrypted object from s3.
      $ {{.HelpName}} --encrypt-key "s3/ferenginar/=32byteslongsecretkeymustbegiven1" s3/ferenginar/1999/old-backup.tgz

`,
}

// Structured message depending on the type of console.
type rmMessage struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

// Colorized message for console printing.
func (r rmMessage) String() string {
	return console.Colorize("Remove", fmt.Sprintf("Removing `%s`.", r.Key))
}

// JSON'ified message for scripting.
func (r rmMessage) JSON() string {
	msgBytes, e := json.Marshal(r)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate command line arguments.
func checkRmSyntax(ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	// Set command flags from context.
	isForce := ctx.Bool("force")
	isRecursive := ctx.Bool("recursive")
	isStdin := ctx.Bool("stdin")
	isDangerous := ctx.Bool("dangerous")
	isNamespaceRemoval := false

	for _, url := range ctx.Args() {
		// clean path for aliases like s3/.
		//Note: UNC path using / works properly in go 1.9.2 even though it breaks the UNC specification.
		url = filepath.ToSlash(filepath.Clean(url))
		// namespace removal applies only for non FS. So filter out if passed url represents a directory
		if !isAliasURLDir(url, encKeyDB) {
			_, path := url2Alias(url)
			isNamespaceRemoval = (path == "")
			break
		}
	}
	if !ctx.Args().Present() && !isStdin {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rm", exitCode)
	}

	// For all recursive operations make sure to check for 'force' flag.
	if (isRecursive || isStdin) && !isForce {
		if isNamespaceRemoval {
			fatalIf(errDummy().Trace(),
				"This operation results in site-wide removal of buckets and objects. If you are really sure, retry this command with ‘--dangerous’ and ‘--force’ flags.")
		}
		fatalIf(errDummy().Trace(),
			"Removal requires --force flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
	}
	if (isRecursive || isStdin) && isNamespaceRemoval && !isDangerous {
		fatalIf(errDummy().Trace(),
			"This operation results in site-wide removal of buckets and objects. If you are really sure, retry this command with ‘--dangerous’ and ‘--force’ flags.")
	}
}

func removeSingle(url string, isIncomplete bool, isFake bool, olderThan int, newerThan int, encKeyDB map[string][]prefixSSEPair) error {
	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Invalid argument `"+url+"`.")
		return exitStatus(globalErrorExitStatus) // End of journey.
	}
	isFetchMeta := true
	alias, _ := url2Alias(url)
	sseKey := getSSEKey(url, encKeyDB[alias])
	content, pErr := clnt.Stat(isIncomplete, isFetchMeta, sseKey)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to remove `"+url+"`.")
		return exitStatus(globalErrorExitStatus)
	}

	// Skip objects older than older--than parameter if specified
	if olderThan > 0 && isOlder(content, olderThan) {
		return nil
	}

	// Skip objects older than older--than parameter if specified
	if newerThan > 0 && isNewer(content, newerThan) {
		return nil
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
				errorIf(pErr.Trace(url), "Failed to remove `"+url+"`.")
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

func removeRecursive(url string, isIncomplete bool, isFake bool, olderThan int, newerThan int, encKeyDB map[string][]prefixSSEPair) error {
	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to remove `"+url+"` recursively.")
		return exitStatus(globalErrorExitStatus) // End of journey.
	}
	contentCh := make(chan *clientContent)
	errorCh := clnt.Remove(isIncomplete, contentCh)

	isEmpty := true
	isRecursive := true
	for content := range clnt.List(isRecursive, isIncomplete, DirLast) {
		isEmpty = false
		if content.Err != nil {
			errorIf(content.Err.Trace(url), "Failed to remove `"+url+"` recursively.")
			switch content.Err.ToGoError().(type) {
			case PathInsufficientPermission:
				// Ignore Permission error.
				continue
			}
			close(contentCh)
			return exitStatus(globalErrorExitStatus)
		}
		urlString := content.URL.Path

		// Skip objects older than --older-than parameter if specified
		if olderThan > 0 && isOlder(content, olderThan) {
			continue
		}

		// Skip objects newer than --newer-than parameter if specified
		if newerThan > 0 && isNewer(content, newerThan) {
			continue
		}

		// list internally mimics recursive directory listing of object prefixes for s3 similar to FS.
		// The rmMessage needs to be printed only for actual objects being deleted and not prefixes.
		if !(content.Time.IsZero() && strings.HasSuffix(urlString, string(filepath.Separator))) {
			printMsg(rmMessage{
				Key:  targetAlias + urlString,
				Size: content.Size,
			})
		}
		if !isFake {
			sent := false
			for !sent {
				select {
				case contentCh <- content:
					sent = true
				case pErr := <-errorCh:
					errorIf(pErr.Trace(urlString), "Failed to remove `"+urlString+"`.")
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
		errorIf(pErr.Trace(url), "Failed to remove `"+url+"` recursively.")
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
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'rm' cli arguments.
	checkRmSyntax(ctx, encKeyDB)

	// rm specific flags.
	isIncomplete := ctx.Bool("incomplete")
	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")
	isStdin := ctx.Bool("stdin")
	olderThan := ctx.Int("older-than")
	newerThan := ctx.Int("newer-than")

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	var rerr error
	var e error
	// Support multiple targets.
	for _, url := range ctx.Args() {
		if isRecursive {
			e = removeRecursive(url, isIncomplete, isFake, olderThan, newerThan, encKeyDB)
		} else {
			e = removeSingle(url, isIncomplete, isFake, olderThan, newerThan, encKeyDB)
		}

		if rerr == nil {
			rerr = e
		}
	}

	if !isStdin {
		return rerr
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := scanner.Text()
		if isRecursive {
			e = removeRecursive(url, isIncomplete, isFake, olderThan, newerThan, encKeyDB)
		} else {
			e = removeSingle(url, isIncomplete, isFake, olderThan, newerThan, encKeyDB)
		}

		if rerr == nil {
			rerr = e
		}
	}

	return rerr
}
