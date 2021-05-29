// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
)

// rm specific flags.
var (
	rmFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "versions",
			Usage: "remove object(s) and all its versions",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "roll back object(s) to current version at specified time",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "delete a specific version of an object",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "remove recursively",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "allow a recursive remove operation",
		},
		cli.BoolFlag{
			Name:  "dangerous",
			Usage: "allow site-wide removal of objects",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "remove incomplete uploads",
		},
		cli.BoolFlag{
			Name:  "fake",
			Usage: "perform a fake remove operation",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "read object names from STDIN",
		},
		cli.StringFlag{
			Name:  "older-than",
			Usage: "remove objects older than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "newer-than",
			Usage: "remove objects newer than L days, M hours and N minutes",
		},
		cli.BoolFlag{
			Name:  "bypass",
			Usage: "bypass governance",
		},
	}
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:         "rm",
	Usage:        "remove objects",
	Action:       mainRm,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(rmFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
  MC_ENCRYPT_KEY: list of comma delimited prefix=secret values

EXAMPLES:
  01. Remove a file.
      {{.Prompt}} {{.HelpName}} 1999/old-backup.tgz

  02. Perform a fake remove operation.
      {{.Prompt}} {{.HelpName}} --fake 1999/old-backup.tgz

  03. Remove all objects recursively from bucket 'jazz-songs' matching the prefix 'louis'.
      {{.Prompt}} {{.HelpName}} --recursive --force s3/jazz-songs/louis/

  04. Remove all objects older than '90' days recursively from bucket 'jazz-songs' matching the prefix 'louis'.
      {{.Prompt}} {{.HelpName}} --recursive --force --older-than 90d s3/jazz-songs/louis/

  05. Remove all objects newer than 7 days and 10 hours recursively from bucket 'pop-songs'
      {{.Prompt}} {{.HelpName}} --recursive --force --newer-than 7d10h s3/pop-songs/

  06. Remove all objects read from STDIN.
      {{.Prompt}} {{.HelpName}} --force --stdin

  07. Remove all objects recursively from Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive --force --dangerous s3

  08. Remove all objects older than '90' days recursively under all buckets.
      {{.Prompt}} {{.HelpName}} --recursive --dangerous --force --older-than 90d s3

  09. Drop all incomplete uploads on the bucket 'jazz-songs'.
      {{.Prompt}} {{.HelpName}} --incomplete --recursive --force s3/jazz-songs/

  10. Remove an encrypted object from Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --encrypt-key "s3/sql-backups/=32byteslongsecretkeymustbegiven1" s3/sql-backups/1999/old-backup.tgz

  11. Bypass object retention in governance mode and delete the object.
      {{.Prompt}} {{.HelpName}} --bypass s3/pop-songs/

  12. Remove a particular version ID.
      {{.Prompt}} {{.HelpName}} s3/docs/money.xls --version-id "f20f3792-4bd4-4288-8d3c-b9d05b3b62f6"

  13. Remove all object versions older than one year.
      {{.Prompt}} {{.HelpName}} s3/docs/ --recursive --versions --rewind 365d

`,
}

// Structured message depending on the type of console.
type rmMessage struct {
	Status    string    `json:"status"`
	Key       string    `json:"key"`
	VersionID string    `json:"versionID"`
	ModTime   time.Time `json:"modTime"`
	Size      int64     `json:"size"`
}

// Colorized message for console printing.
func (r rmMessage) String() string {
	msg := console.Colorize("Remove", fmt.Sprintf("Removing `%s`", r.Key))
	if r.VersionID != "" {
		if !r.ModTime.IsZero() {
			msg += fmt.Sprintf(" (versionId=%s, modTime=%s)", r.VersionID, r.ModTime)
		} else {
			msg += fmt.Sprintf(" (versionId=%s)", r.VersionID)
		}
	}
	msg += "."
	return msg
}

// JSON'ified message for scripting.
func (r rmMessage) JSON() string {
	r.Status = "success"
	msgBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate command line arguments.
func checkRmSyntax(ctx context.Context, cliCtx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	// Set command flags from context.
	isForce := cliCtx.Bool("force")
	isRecursive := cliCtx.Bool("recursive")
	isStdin := cliCtx.Bool("stdin")
	isDangerous := cliCtx.Bool("dangerous")
	isVersions := cliCtx.Bool("versions")
	versionID := cliCtx.String("version-id")
	rewind := cliCtx.String("rewind")
	isNamespaceRemoval := false

	if versionID != "" && (isRecursive || isVersions || rewind != "") {
		fatalIf(errDummy().Trace(),
			"You cannot specify --version-id with any of --versions, --rewind and --recursive flags.")
	}

	for _, url := range cliCtx.Args() {
		// clean path for aliases like s3/.
		// Note: UNC path using / works properly in go 1.9.2 even though it breaks the UNC specification.
		url = filepath.ToSlash(filepath.Clean(url))
		// namespace removal applies only for non FS. So filter out if passed url represents a directory
		dir := isAliasURLDir(ctx, url, encKeyDB, time.Time{})
		if !dir {
			_, path := url2Alias(url)
			isNamespaceRemoval = (path == "")
			break
		}
		if dir && isRecursive && !isForce {
			fatalIf(errDummy().Trace(),
				"Removal requires --force flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
		}
		if dir && !isRecursive {
			fatalIf(errDummy().Trace(),
				"Removal requires --recursive flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
		}
	}
	if !cliCtx.Args().Present() && !isStdin {
		exitCode := 1
		cli.ShowCommandHelpAndExit(cliCtx, "rm", exitCode)
	}

	// For all recursive or versions bulk deletion operations make sure to check for 'force' flag.
	if (isVersions || isRecursive || isStdin) && !isForce {
		if isNamespaceRemoval {
			fatalIf(errDummy().Trace(),
				"This operation results in site-wide removal of objects. If you are really sure, retry this command with ‘--dangerous’ and ‘--force’ flags.")
		}
		fatalIf(errDummy().Trace(),
			"Removal requires --force flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
	}
	if (isRecursive || isStdin) && isNamespaceRemoval && !isDangerous {
		fatalIf(errDummy().Trace(),
			"This operation results in site-wide removal of objects. If you are really sure, retry this command with ‘--dangerous’ and ‘--force’ flags.")
	}
}

// Remove a single object or a single version in a versioned bucket
func removeSingle(url, versionID string, isIncomplete, isFake, isForce, isBypass bool, olderThan, newerThan string, encKeyDB map[string][]prefixSSEPair) error {
	ctx, cancel := context.WithCancel(globalContext)
	defer cancel()

	var (
		// A HEAD request can fail with:
		// - 400 Bad Request when the object SSE-C
		// - 405 Method Not Allowed  when this is a delete marker
		// In those cases, we still want t remove the target object/version
		// so we simply ignore them.
		ignoreStatError bool

		isDir   bool
		size    int64
		modTime time.Time
	)

	_, content, pErr := url2Stat(ctx, url, versionID, false, encKeyDB, time.Time{})
	if pErr != nil {
		switch minio.ToErrorResponse(pErr.ToGoError()).StatusCode {
		case http.StatusBadRequest, http.StatusMethodNotAllowed:
			ignoreStatError = true
		default:
			errorIf(pErr.Trace(url), "Failed to remove `"+url+"`.")
			return exitStatus(globalErrorExitStatus)
		}
	} else {
		isDir = content.Type.IsDir()
		size = content.Size
		modTime = content.Time
	}

	// We should not proceed
	if ignoreStatError && olderThan != "" || newerThan != "" {
		errorIf(pErr.Trace(url), "Unable to stat `"+url+"`.")
		return exitStatus(globalErrorExitStatus)
	}

	// Skip objects older than older--than parameter if specified
	if olderThan != "" && isOlder(modTime, olderThan) {
		return nil
	}

	// Skip objects older than older--than parameter if specified
	if newerThan != "" && isNewer(modTime, newerThan) {
		return nil
	}

	printMsg(rmMessage{
		Key:       url,
		Size:      size,
		VersionID: versionID,
	})

	if !isFake {
		targetAlias, targetURL, _ := mustExpandAlias(url)
		clnt, pErr := newClientFromAlias(targetAlias, targetURL)
		if pErr != nil {
			errorIf(pErr.Trace(url), "Invalid argument `"+url+"`.")
			return exitStatus(globalErrorExitStatus) // End of journey.
		}

		if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) && isDir {
			targetURL = targetURL + string(clnt.GetURL().Separator)
		}

		contentCh := make(chan *ClientContent, 1)
		contentCh <- &ClientContent{URL: *newClientURL(targetURL), VersionID: versionID}
		close(contentCh)
		isRemoveBucket := false
		errorCh := clnt.Remove(ctx, isIncomplete, isRemoveBucket, isBypass, contentCh)
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

// listAndRemove uses listing before removal, it can list recursively or not, with versions or not.
//   Use cases:
//      * Remove objects recursively
//      * Remove all versions of a single object
func listAndRemove(url string, timeRef time.Time, withVersions, isRecursive, isIncomplete, isFake, isBypass bool, olderThan, newerThan string, encKeyDB map[string][]prefixSSEPair) error {
	ctx, cancelRemove := context.WithCancel(globalContext)
	defer cancelRemove()

	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to remove `"+url+"` recursively.")
		return exitStatus(globalErrorExitStatus) // End of journey.
	}
	contentCh := make(chan *ClientContent)
	isRemoveBucket := false

	errorCh := clnt.Remove(ctx, isIncomplete, isRemoveBucket, isBypass, contentCh)

	listOpts := ListOptions{Recursive: isRecursive, Incomplete: isIncomplete, ShowDir: DirLast}
	if !timeRef.IsZero() {
		listOpts.WithOlderVersions = withVersions
		listOpts.WithDeleteMarkers = true
		listOpts.TimeRef = timeRef
	}

	atLeastOneObjectFound := false

	for content := range clnt.List(ctx, listOpts) {
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

		// rm command is not supposed to remove buckets, ignore if this is a bucket name
		if content.URL.Type == objectStorage && strings.LastIndex(urlString, string(content.URL.Separator)) == 0 {
			continue
		}

		if !isRecursive {
			currentObjectURL := targetAlias + getKey(content)
			standardizedURL := getStandardizedURL(currentObjectURL)
			if !strings.HasPrefix(url, standardizedURL) {
				break
			}
		}

		// This will mark that we found at least one target object
		// even that it could be ineligible for deletion. So we can
		// inform the user that he was searching in an empty area
		atLeastOneObjectFound = true

		if !content.Time.IsZero() {
			// Skip objects older than --older-than parameter, if specified
			if olderThan != "" && isOlder(content.Time, olderThan) {
				continue
			}

			// Skip objects newer than --newer-than parameter if specified
			if newerThan != "" && isNewer(content.Time, newerThan) {
				continue
			}
		} else {
			// Skip prefix levels.
			continue
		}

		printMsg(rmMessage{
			Key:       targetAlias + urlString,
			Size:      content.Size,
			VersionID: content.VersionID,
			ModTime:   content.Time,
		})

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

	if !atLeastOneObjectFound {
		errorIf(errDummy().Trace(url), "No object/version found to be removed in `"+url+"`.")
		return exitStatus(globalErrorExitStatus)
	}

	return nil
}

// main for rm command.
func mainRm(cliCtx *cli.Context) error {
	ctx, cancelRm := context.WithCancel(globalContext)
	defer cancelRm()

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'rm' cli arguments.
	checkRmSyntax(ctx, cliCtx, encKeyDB)

	// rm specific flags.
	isIncomplete := cliCtx.Bool("incomplete")
	isRecursive := cliCtx.Bool("recursive")
	isFake := cliCtx.Bool("fake")
	isStdin := cliCtx.Bool("stdin")
	isBypass := cliCtx.Bool("bypass")
	olderThan := cliCtx.String("older-than")
	newerThan := cliCtx.String("newer-than")
	isForce := cliCtx.Bool("force")
	withVersions := cliCtx.Bool("versions")
	versionID := cliCtx.String("version-id")
	rewind := parseRewindFlag(cliCtx.String("rewind"))

	if withVersions && rewind.IsZero() {
		rewind = time.Now().UTC()
	}

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	var rerr error
	var e error
	// Support multiple targets.
	for _, url := range cliCtx.Args() {
		if isRecursive || withVersions {
			e = listAndRemove(url, rewind, withVersions, isRecursive, isIncomplete, isFake, isBypass, olderThan, newerThan, encKeyDB)
		} else {
			e = removeSingle(url, versionID, isIncomplete, isFake, isForce, isBypass, olderThan, newerThan, encKeyDB)
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
		if isRecursive || withVersions {
			e = listAndRemove(url, rewind, withVersions, isRecursive, isIncomplete, isFake, isBypass, olderThan, newerThan, encKeyDB)
		} else {
			e = removeSingle(url, versionID, isIncomplete, isFake, isForce, isBypass, olderThan, newerThan, encKeyDB)
		}
		if rerr == nil {
			rerr = e
		}
	}

	return rerr
}
