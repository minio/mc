// Copyright (c) 2015-2022 MinIO, Inc.
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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/pkg/v3/console"
)

// rm specific flags.
var (
	rmFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "versions",
			Usage: "remove object(s) and all its versions",
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
		cli.StringFlag{
			Name:  "rewind",
			Usage: "roll back object(s) to current version at specified time",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "delete a specific version of an object",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "remove incomplete uploads",
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "perform a fake remove operation",
		},
		cli.BoolFlag{
			Name:   "fake",
			Usage:  "perform a fake remove operation",
			Hidden: true, // deprecated 2022
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "read object names from STDIN",
		},
		cli.StringFlag{
			Name:  "older-than",
			Usage: "remove objects older than value in duration string (e.g. 7d10h31s)",
		},
		cli.StringFlag{
			Name:  "newer-than",
			Usage: "remove objects newer than value in duration string (e.g. 7d10h31s)",
		},
		cli.BoolFlag{
			Name:  "bypass",
			Usage: "bypass governance",
		},
		cli.BoolFlag{
			Name:  "non-current",
			Usage: "remove object(s) versions that are non-current",
		},
		cli.BoolFlag{
			Name:   "purge",
			Usage:  "attempt a prefix purge, requires confirmation please use with caution - only works with '--force'",
			Hidden: true,
		},
	}
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:         "rm",
	Usage:        "remove object(s)",
	Action:       mainRm,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(rmFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  01. Remove a file.
      {{.Prompt}} {{.HelpName}} 1999/old-backup.tgz

  02. Perform a fake remove operation.
      {{.Prompt}} {{.HelpName}} --dry-run 1999/old-backup.tgz

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

  10. Bypass object retention in governance mode and delete the object.
      {{.Prompt}} {{.HelpName}} --bypass s3/pop-songs/

  11. Remove a particular version ID.
      {{.Prompt}} {{.HelpName}} s3/docs/money.xls --version-id "f20f3792-4bd4-4288-8d3c-b9d05b3b62f6"

  12. Remove all object versions older than one year.
      {{.Prompt}} {{.HelpName}} s3/docs/ --recursive --versions --rewind 365d

  14. Perform a fake removal of object(s) versions that are non-current and older than 10 days. If top-level version is a delete 
  marker, this will also be deleted when --non-current flag is specified.
      {{.Prompt}} {{.HelpName}} s3/docs/ --recursive --force --versions --non-current --older-than 10d --dry-run
`,
}

// Structured message depending on the type of console.
type rmMessage struct {
	Status       string     `json:"status"`
	Key          string     `json:"key"`
	DeleteMarker bool       `json:"deleteMarker"`
	VersionID    string     `json:"versionID"`
	ModTime      *time.Time `json:"modTime"`
	DryRun       bool       `json:"dryRun"`
}

// Colorized message for console printing.
func (r rmMessage) String() string {
	msg := "Removed "
	if r.DryRun {
		msg = "DRYRUN: Removing "
	}

	if r.DeleteMarker {
		msg = "Created delete marker "
	}

	msg += console.Colorize("Removed", fmt.Sprintf("`%s`", r.Key))
	if r.VersionID != "" {
		msg += fmt.Sprintf(" (versionId=%s)", r.VersionID)
		if r.ModTime != nil {
			msg += fmt.Sprintf(" (modTime=%s)", r.ModTime.Format(printDate))
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
func checkRmSyntax(ctx context.Context, cliCtx *cli.Context) {
	// Set command flags from context.
	isForce := cliCtx.Bool("force")
	isRecursive := cliCtx.Bool("recursive")
	isStdin := cliCtx.Bool("stdin")
	isDangerous := cliCtx.Bool("dangerous")
	isVersions := cliCtx.Bool("versions")
	isNoncurrentVersion := cliCtx.Bool("non-current")
	isForceDel := cliCtx.Bool("purge")
	versionID := cliCtx.String("version-id")
	rewind := cliCtx.String("rewind")
	isNamespaceRemoval := false

	if versionID != "" && (isRecursive || isVersions || rewind != "") {
		fatalIf(errDummy().Trace(),
			"You cannot specify --version-id with any of --versions, --rewind and --recursive flags.")
	}

	if isNoncurrentVersion && (!isVersions || !isRecursive) {
		fatalIf(errDummy().Trace(),
			"You cannot specify --non-current without --versions --recursive, please use --non-current --versions --recursive.")
	}

	if isForceDel && !isForce {
		fatalIf(errDummy().Trace(),
			"You cannot specify --purge without --force.")
	}

	if isForceDel && isRecursive {
		fatalIf(errDummy().Trace(),
			"You cannot specify --purge with --recursive.")
	}

	if isForceDel && (isNoncurrentVersion || isVersions || cliCtx.IsSet("older-than") || cliCtx.IsSet("newer-than") || versionID != "") {
		fatalIf(errDummy().Trace(),
			"You cannot specify --purge flag with any flag(s) other than --force.")
	}

	if !isForceDel {
		for _, url := range cliCtx.Args() {
			// clean path for aliases like s3/.
			// Note: UNC path using / works properly in go 1.9.2 even though it breaks the UNC specification.
			url = filepath.ToSlash(filepath.Clean(url))
			// namespace removal applies only for non FS. So filter out if passed url represents a directory
			dir, _ := isAliasURLDir(ctx, url, nil, time.Time{}, false)
			if dir {
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
	}

	if !cliCtx.Args().Present() && !isStdin {
		exitCode := 1
		showCommandHelpAndExit(cliCtx, exitCode)
	}

	// For all recursive or versions bulk deletion operations make sure to check for 'force' flag.
	if (isVersions || isRecursive || isStdin) && !isForce {
		fatalIf(errDummy().Trace(),
			"Removal requires --force flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
	}

	if isNamespaceRemoval && (!isDangerous || !isForce) {
		fatalIf(errDummy().Trace(),
			"This operation results in site-wide removal of objects. If you are really sure, retry this command with ‘--dangerous’ and ‘--force’ flags.")
	}
}

// Remove a single object or a single version in a versioned bucket
func removeSingle(url, versionID string, opts removeOpts) error {
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
		modTime time.Time
	)

	targetAlias, targetURL, _ := mustExpandAlias(url)
	if !opts.isForceDel {
		_, content, pErr := url2Stat(ctx, url2StatOptions{
			urlStr:                  url,
			versionID:               versionID,
			fileAttr:                false,
			timeRef:                 time.Time{},
			isZip:                   false,
			ignoreBucketExistsCheck: false,
		})
		if pErr != nil {
			switch st := minio.ToErrorResponse(pErr.ToGoError()).StatusCode; st {
			case http.StatusBadRequest, http.StatusMethodNotAllowed:
				ignoreStatError = true
			default:
				_, ok := pErr.ToGoError().(ObjectMissing)
				ignoreStatError = (st == http.StatusServiceUnavailable || ok || st == http.StatusNotFound) && (opts.isForce && opts.isForceDel)
				if !ignoreStatError {
					errorIf(pErr.Trace(url), "Failed to remove `%s`.", url)
					return exitStatus(globalErrorExitStatus)
				}
			}
		} else {
			isDir = content.Type.IsDir()
			modTime = content.Time
		}

		// We should not proceed
		if ignoreStatError && (opts.olderThan != "" || opts.newerThan != "") {
			errorIf(pErr.Trace(url), "Unable to stat `%s`.", url)
			return exitStatus(globalErrorExitStatus)
		}

		// Skip objects older than older--than parameter if specified
		if opts.olderThan != "" && isOlder(modTime, opts.olderThan) {
			return nil
		}

		// Skip objects older than older--than parameter if specified
		if opts.newerThan != "" && isNewer(modTime, opts.newerThan) {
			return nil
		}

		if opts.isFake {
			printDryRunMsg(targetAlias, content, opts.withVersions)
			return nil
		}
	}

	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Invalid argument `%s`.", url)
		return exitStatus(globalErrorExitStatus) // End of journey.
	}

	if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) && isDir {
		targetURL = targetURL + string(clnt.GetURL().Separator)
	}

	contentCh := make(chan *ClientContent, 1)
	contentURL := *newClientURL(targetURL)
	contentCh <- &ClientContent{URL: contentURL, VersionID: versionID}
	close(contentCh)
	isRemoveBucket := false
	resultCh := clnt.Remove(ctx, opts.isIncomplete, isRemoveBucket, opts.isBypass, opts.isForce && opts.isForceDel, contentCh)
	for result := range resultCh {
		if result.Err != nil {
			errorIf(result.Err.Trace(url), "Failed to remove `%s`.", url)
			switch result.Err.ToGoError().(type) {
			case PathInsufficientPermission:
				// Ignore Permission error.
				continue
			}
			return exitStatus(globalErrorExitStatus)
		}
		msg := rmMessage{
			Key:       path.Join(targetAlias, result.BucketName, result.ObjectName),
			VersionID: result.ObjectVersionID,
		}
		if result.DeleteMarker {
			msg.DeleteMarker = true
			msg.VersionID = result.DeleteMarkerVersionID
		}
		printMsg(msg)
	}
	return nil
}

type removeOpts struct {
	timeRef           time.Time
	withVersions      bool
	nonCurrentVersion bool
	isForce           bool
	isRecursive       bool
	isIncomplete      bool
	isFake            bool
	isBypass          bool
	isForceDel        bool
	olderThan         string
	newerThan         string
}

func printDryRunMsg(targetAlias string, content *ClientContent, printModTime bool) {
	if content == nil {
		return
	}
	msg := rmMessage{
		Status:    "success",
		DryRun:    true,
		Key:       targetAlias + getKey(content),
		VersionID: content.VersionID,
	}
	if printModTime {
		msg.ModTime = &content.Time
	}
	printMsg(msg)
}

// listAndRemove uses listing before removal, it can list recursively or not, with versions or not.
//
//	Use cases:
//	   * Remove objects recursively
//	   * Remove all versions of a single object
func listAndRemove(url string, opts removeOpts) error {
	ctx, cancelRemove := context.WithCancel(globalContext)
	defer cancelRemove()

	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to remove `%s` recursively.", url)
		return exitStatus(globalErrorExitStatus) // End of journey.
	}
	contentCh := make(chan *ClientContent)
	isRemoveBucket := false

	listOpts := ListOptions{Recursive: opts.isRecursive, Incomplete: opts.isIncomplete, ShowDir: DirLast}
	if !opts.timeRef.IsZero() {
		listOpts.WithOlderVersions = opts.withVersions
		listOpts.WithDeleteMarkers = true
		listOpts.TimeRef = opts.timeRef
	}
	atLeastOneObjectFound := false

	resultCh := clnt.Remove(ctx, opts.isIncomplete, isRemoveBucket, opts.isBypass, false, contentCh)

	var lastPath string
	var perObjectVersions []*ClientContent
	for content := range clnt.List(ctx, listOpts) {
		if content.Err != nil {
			errorIf(content.Err.Trace(url), "Failed to remove `%s` recursively.", url)
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

		if !opts.isRecursive {
			currentObjectURL := getStandardizedURL(targetAlias + getKey(content))
			standardizedURL := getStandardizedURL(currentObjectURL)
			if !strings.HasPrefix(url, standardizedURL) {
				break
			}
		}

		if opts.nonCurrentVersion && opts.isRecursive && opts.withVersions {
			if lastPath != content.URL.Path {
				lastPath = content.URL.Path
				for _, content := range perObjectVersions {
					if content.IsLatest && !content.IsDeleteMarker {
						continue
					}
					if !content.Time.IsZero() {
						// Skip objects older than --older-than parameter, if specified
						if opts.olderThan != "" && isOlder(content.Time, opts.olderThan) {
							continue
						}

						// Skip objects newer than --newer-than parameter if specified
						if opts.newerThan != "" && isNewer(content.Time, opts.newerThan) {
							continue
						}
					} else {
						// Skip prefix levels.
						continue
					}

					if opts.isFake {
						printDryRunMsg(targetAlias, content, true)
						continue
					}

					sent := false
					for !sent {
						select {
						case contentCh <- content:
							sent = true
						case result := <-resultCh:
							path := path.Join(targetAlias, result.BucketName, result.ObjectName)
							if result.Err != nil {
								errorIf(result.Err.Trace(path),
									"Failed to remove `%s`.", path)
								switch result.Err.ToGoError().(type) {
								case PathInsufficientPermission:
									// Ignore Permission error.
									continue
								}
								close(contentCh)
								return exitStatus(globalErrorExitStatus)
							}
							msg := rmMessage{
								Key:       path,
								VersionID: result.ObjectVersionID,
							}
							if result.DeleteMarker {
								msg.DeleteMarker = true
								msg.VersionID = result.DeleteMarkerVersionID
							}
							printMsg(msg)
						}
					}
				}
				perObjectVersions = []*ClientContent{}
			}
			atLeastOneObjectFound = true
			perObjectVersions = append(perObjectVersions, content)
			continue
		}

		// This will mark that we found at least one target object
		// even that it could be ineligible for deletion. So we can
		// inform the user that he was searching in an empty area
		atLeastOneObjectFound = true

		if !content.Time.IsZero() {
			// Skip objects older than --older-than parameter, if specified
			if opts.olderThan != "" && isOlder(content.Time, opts.olderThan) {
				continue
			}

			// Skip objects newer than --newer-than parameter if specified
			if opts.newerThan != "" && isNewer(content.Time, opts.newerThan) {
				continue
			}
		} else {
			// Skip prefix levels.
			continue
		}

		if !opts.isFake {
			sent := false
			for !sent {
				select {
				case contentCh <- content:
					sent = true
				case result := <-resultCh:
					path := path.Join(targetAlias, result.BucketName, result.ObjectName)
					if result.Err != nil {
						errorIf(result.Err.Trace(path),
							"Failed to remove `%s`.", path)
						switch e := result.Err.ToGoError().(type) {
						case PathInsufficientPermission:
							// Ignore Permission error.
							continue
						case minio.ErrorResponse:
							if strings.Contains(e.Message, "Object is WORM protected and cannot be overwritten") {
								continue
							}
						}
						close(contentCh)
						return exitStatus(globalErrorExitStatus)
					}
					msg := rmMessage{
						Key:       path,
						VersionID: result.ObjectVersionID,
					}
					if result.DeleteMarker {
						msg.DeleteMarker = true
						msg.VersionID = result.DeleteMarkerVersionID
					}
					printMsg(msg)
				}
			}
		} else {
			printDryRunMsg(targetAlias, content, opts.withVersions)
		}
	}

	if opts.nonCurrentVersion && opts.isRecursive && opts.withVersions {
		for _, content := range perObjectVersions {
			if content.IsLatest && !content.IsDeleteMarker {
				continue
			}
			if !content.Time.IsZero() {
				// Skip objects older than --older-than parameter, if specified
				if opts.olderThan != "" && isOlder(content.Time, opts.olderThan) {
					continue
				}

				// Skip objects newer than --newer-than parameter if specified
				if opts.newerThan != "" && isNewer(content.Time, opts.newerThan) {
					continue
				}
			} else {
				// Skip prefix levels.
				continue
			}

			if opts.isFake {
				printDryRunMsg(targetAlias, content, true)
				continue
			}

			sent := false
			for !sent {
				select {
				case contentCh <- content:
					sent = true
				case result := <-resultCh:
					path := path.Join(targetAlias, result.BucketName, result.ObjectName)
					if result.Err != nil {
						errorIf(result.Err.Trace(path),
							"Failed to remove `%s`.", path)
						switch result.Err.ToGoError().(type) {
						case PathInsufficientPermission:
							// Ignore Permission error.
							continue
						}
						close(contentCh)
						return exitStatus(globalErrorExitStatus)
					}
					msg := rmMessage{
						Key:       path,
						VersionID: result.ObjectVersionID,
					}
					if result.DeleteMarker {
						msg.DeleteMarker = true
						msg.VersionID = result.DeleteMarkerVersionID
					}
					printMsg(msg)
				}
			}
		}
	}

	close(contentCh)
	if opts.isFake {
		return nil
	}
	for result := range resultCh {
		path := path.Join(targetAlias, result.BucketName, result.ObjectName)
		if result.Err != nil {
			errorIf(result.Err.Trace(path), "Failed to remove `%s` recursively.", path)
			switch result.Err.ToGoError().(type) {
			case PathInsufficientPermission:
				// Ignore Permission error.
				continue
			}
			return exitStatus(globalErrorExitStatus)
		}
		msg := rmMessage{
			Key:       path,
			VersionID: result.ObjectVersionID,
		}
		if result.DeleteMarker {
			msg.DeleteMarker = true
			msg.VersionID = result.DeleteMarkerVersionID
		}
		printMsg(msg)
	}

	if !atLeastOneObjectFound {
		if opts.isForce {
			// Do not throw an exit code with --force check unix `rm -f`
			// behavior and do not print an error as well.
			return nil
		}
		errorIf(errDummy().Trace(url), "No object/version found to be removed in `%s`.", url)
		return exitStatus(globalErrorExitStatus)
	}

	return nil
}

// main for rm command.
func mainRm(cliCtx *cli.Context) error {
	ctx, cancelRm := context.WithCancel(globalContext)
	defer cancelRm()

	checkRmSyntax(ctx, cliCtx)

	isIncomplete := cliCtx.Bool("incomplete")
	isRecursive := cliCtx.Bool("recursive")
	isFake := cliCtx.Bool("dry-run") || cliCtx.Bool("fake")
	isStdin := cliCtx.Bool("stdin")
	isBypass := cliCtx.Bool("bypass")
	olderThan := cliCtx.String("older-than")
	newerThan := cliCtx.String("newer-than")
	isForce := cliCtx.Bool("force")
	isForceDel := cliCtx.Bool("purge")
	withNoncurrentVersion := cliCtx.Bool("non-current")
	withVersions := cliCtx.Bool("versions")
	versionID := cliCtx.String("version-id")
	rewind := parseRewindFlag(cliCtx.String("rewind"))

	if withVersions && rewind.IsZero() {
		rewind = time.Now().UTC()
	}

	// Set color.
	console.SetColor("Removed", color.New(color.FgGreen, color.Bold))

	var rerr error
	var e error
	// Support multiple targets.
	for _, url := range cliCtx.Args() {
		if isRecursive || withVersions {
			e = listAndRemove(url, removeOpts{
				timeRef:           rewind,
				withVersions:      withVersions,
				nonCurrentVersion: withNoncurrentVersion,
				isForce:           isForce,
				isRecursive:       isRecursive,
				isIncomplete:      isIncomplete,
				isFake:            isFake,
				isBypass:          isBypass,
				olderThan:         olderThan,
				newerThan:         newerThan,
			})
		} else {
			e = removeSingle(url, versionID, removeOpts{
				isIncomplete: isIncomplete,
				isFake:       isFake,
				isForce:      isForce,
				isForceDel:   isForceDel,
				isBypass:     isBypass,
				olderThan:    olderThan,
				newerThan:    newerThan,
			})
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
			e = listAndRemove(url, removeOpts{
				timeRef:           rewind,
				withVersions:      withVersions,
				nonCurrentVersion: withNoncurrentVersion,
				isForce:           isForce,
				isRecursive:       isRecursive,
				isIncomplete:      isIncomplete,
				isFake:            isFake,
				isBypass:          isBypass,
				olderThan:         olderThan,
				newerThan:         newerThan,
			})
		} else {
			e = removeSingle(url, versionID, removeOpts{
				isIncomplete: isIncomplete,
				isFake:       isFake,
				isForce:      isForce,
				isForceDel:   isForceDel,
				isBypass:     isBypass,
				olderThan:    olderThan,
				newerThan:    newerThan,
			})
		}
		if rerr == nil {
			rerr = e
		}
	}

	return rerr
}
