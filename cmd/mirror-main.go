/*
 * Minio Client, (C) 2015, 2016 Minio, Inc.
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
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// mirror specific flags.
var (
	mirrorFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force overwrite of an existing target(s).",
		},
		cli.BoolFlag{
			Name:  "fake",
			Usage: "Perform a fake mirror operation.",
		},
		cli.BoolFlag{
			Name:  "watch, w",
			Usage: "Watch and mirror for changes.",
		},
		cli.BoolFlag{
			Name:  "remove",
			Usage: "Remove extraneous file(s) on target.",
		},
		cli.StringFlag{
			Name:  "region",
			Usage: "Specify which region to select when creating new buckets.",
			Value: "us-east-1",
		},
		cli.BoolFlag{
			Name:  "a",
			Usage: "Preserve bucket policies rules.",
		},
		cli.StringSliceFlag{
			Name:  "exclude",
			Usage: "Exclude the source file/object that matches the passed shell file name pattern.",
		},
	}
)

//  Mirror folders recursively from a single source to many destinations
var mirrorCmd = cli.Command{
	Name:   "mirror",
	Usage:  "Mirror buckets and folders.",
	Action: mainMirror,
	Before: setGlobalsFromContext,
	Flags:  append(mirrorFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
  MC_MULTIPART_THREADS: To set number of multipart threads. By default it is 4.

EXAMPLES:
   1. Mirror a bucket recursively from Minio cloud storage to a bucket on Amazon S3 cloud storage.
      $ {{.HelpName}} play/photos/2014 s3/backup-photos

   2. Mirror a local folder recursively to Amazon S3 cloud storage.
      $ {{.HelpName}} backup/ s3/archive

   3. Mirror a bucket from aliased Amazon S3 cloud storage to a folder on Windows.
      $ {{.HelpName}} s3\documents\2014\ C:\backup\2014

   4. Mirror a bucket from aliased Amazon S3 cloud storage to a local folder use '--force' to overwrite destination.
      $ {{.HelpName}} --force s3/miniocloud miniocloud-backup

   5. Mirror a bucket from Minio cloud storage to a bucket on Amazon S3 cloud storage and remove any extraneous
      files on Amazon S3 cloud storage. NOTE: '--remove' is only supported with '--force'.
      $ {{.HelpName}} --force --remove play/photos/2014 s3/backup-photos/2014

   6. Continuously mirror a local folder recursively to Minio cloud storage. '--watch' continuously watches for
      new objects and uploads them.
      $ {{.HelpName}} --force --remove --watch /var/lib/backups play/backups

   7. Mirror a bucket from aliased Amazon S3 cloud storage to a local folder.
      Exclude all .* files and *.temp files when mirroring.
      $ {{.HelpName}} --exclude ".*" --exclude "*.temp" s3/test ~/test

`,
}

type mirrorJob struct {

	// the channel to trap SIGKILL signals
	trapCh <-chan bool

	// mutex for shutdown, this prevents the shutdown
	// to be initiated multiple times
	m *sync.Mutex

	// channel for errors
	errorCh chan *probe.Error

	// Contains if watcher is currently running.
	watcherRunning bool

	// the global watcher object, which receives notifications of created
	// and deleted files
	watcher *Watcher

	// Hold operation status information
	status  Status
	scanBar scanBarFunc

	// waitgroup for status goroutine, waits till all status
	// messages have been written and received
	wgStatus *sync.WaitGroup
	// channel for status messages
	statusCh chan URLs

	// Store last watch error
	watchErr *probe.Error
	// Store last mirror error
	mirrorErr *probe.Error

	TotalObjects int64
	TotalBytes   int64

	sourceURL string
	targetURL string

	isFake, isForce, isRemove, isWatch bool

	excludeOptions []string
}

// mirrorMessage container for file mirror messages
type mirrorMessage struct {
	Status     string `json:"status"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Size       int64  `json:"size"`
	TotalCount int64  `json:"totalCount"`
	TotalSize  int64  `json:"totalSize"`
}

// String colorized mirror message
func (m mirrorMessage) String() string {
	return console.Colorize("Mirror", fmt.Sprintf("`%s` -> `%s`", m.Source, m.Target))
}

// JSON jsonified mirror message
func (m mirrorMessage) JSON() string {
	m.Status = "success"
	mirrorMessageBytes, e := json.Marshal(m)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(mirrorMessageBytes)
}

// doRemove - removes files on target.
func (mj *mirrorJob) doRemove(sURLs URLs) URLs {
	if mj.isFake {
		return sURLs.WithError(nil)
	}

	// We are not removing incomplete uploads.
	isIncomplete := false

	// Construct proper path with alias.
	targetWithAlias := filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path)

	// Remove extraneous file on target.
	err := probe.NewError(removeSingle(targetWithAlias, isIncomplete, mj.isFake, 0))
	return sURLs.WithError(err)
}

// doMirror - Mirror an object to multiple destination. URLs status contains a copy of sURLs and error if any.
func (mj *mirrorJob) doMirror(sURLs URLs) URLs {

	if sURLs.Error != nil { // Erroneous sURLs passed.
		return sURLs.WithError(sURLs.Error.Trace())
	}

	//s For a fake mirror make sure we update respective progress bars
	// and accounting readers under relevant conditions.
	if mj.isFake {
		mj.status.Add(sURLs.SourceContent.Size)
		return sURLs.WithError(nil)
	}

	sourceAlias := sURLs.SourceAlias
	sourceURL := sURLs.SourceContent.URL
	targetAlias := sURLs.TargetAlias
	targetURL := sURLs.TargetContent.URL
	length := sURLs.SourceContent.Size

	mj.status.SetCaption(sourceURL.String() + ": ")

	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))
	mj.status.PrintMsg(mirrorMessage{
		Source:     sourcePath,
		Target:     targetPath,
		Size:       length,
		TotalCount: sURLs.TotalCount,
		TotalSize:  sURLs.TotalSize,
	})
	return uploadSourceToTargetURL(sURLs, mj.status)
}

// Go routine to update progress status
func (mj *mirrorJob) startStatus() {
	mj.wgStatus.Add(1)

	// wait for status messages on statusChan, show error messages
	go func() {
		// now we want to start the progress bar
		mj.status.Start()
		defer mj.wgStatus.Done()

		for sURLs := range mj.statusCh {
			if sURLs.Error != nil {
				// Save last mirror error
				mj.mirrorErr = sURLs.Error
				// Print in new line and adjust to top so that we
				// don't print over the ongoing progress bar.
				if sURLs.SourceContent != nil {
					errorIf(sURLs.Error.Trace(sURLs.SourceContent.URL.String()),
						fmt.Sprintf("Failed to copy `%s`.", sURLs.SourceContent.URL.String()))
				} else {
					// When sURLs.SourceContent is nil, we know that we have an error related to removing
					errorIf(sURLs.Error.Trace(sURLs.TargetContent.URL.String()),
						fmt.Sprintf("Failed to remove `%s`.", sURLs.TargetContent.URL.String()))
				}
			}

			if sURLs.SourceContent != nil {
			} else if sURLs.TargetContent != nil {
				// Construct user facing message and path.
				targetPath := filepath.ToSlash(filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path))
				mj.status.PrintMsg(rmMessage{Key: targetPath})
			}
		}
	}()
}

// Stop status go routine, further updates could lead to a crash
func (mj *mirrorJob) stopStatus() {
	close(mj.statusCh)
	mj.wgStatus.Wait()
	mj.status.Finish()
}

// this goroutine will watch for notifications, and add modified objects to the queue
func (mj *mirrorJob) watchMirror() {

	for {
		select {
		case event, ok := <-mj.watcher.Events():
			if !ok {
				// channel closed
				return
			}

			// It will change the expanded alias back to the alias
			// again, by replacing the sourceUrlFull with the sourceAlias.
			// This url will be used to mirror.
			sourceAlias, sourceURLFull, _ := mustExpandAlias(mj.sourceURL)

			// If the passed source URL points to fs, fetch the absolute src path
			// to correctly calculate targetPath
			if sourceAlias == "" {
				tmpSrcURL, err := filepath.Abs(sourceURLFull)
				if err == nil {
					sourceURLFull = tmpSrcURL
				}
			}

			sourceURL := newClientURL(event.Path)
			aliasedPath := strings.Replace(event.Path, sourceURLFull, mj.sourceURL, -1)

			// build target path, it is the relative of the event.Path with the sourceUrl
			// joined to the targetURL.
			sourceSuffix := strings.TrimPrefix(event.Path, sourceURLFull)
			targetPath := urlJoinPath(mj.targetURL, sourceSuffix)

			// newClient needs the unexpanded  path, newCLientURL needs the expanded path
			targetAlias, expandedTargetPath, _ := mustExpandAlias(targetPath)
			targetURL := newClientURL(expandedTargetPath)

			if event.Type == EventCreate {
				// we are checking if a destination file exists now, and if we only
				// overwrite it when force is enabled.
				mirrorURL := URLs{
					SourceAlias:   sourceAlias,
					SourceContent: &clientContent{URL: *sourceURL},
					TargetAlias:   targetAlias,
					TargetContent: &clientContent{URL: *targetURL},
				}
				if event.Size == 0 {
					sourceClient, err := newClient(aliasedPath)
					if err != nil {
						// cannot create sourceclient
						mj.statusCh <- mirrorURL.WithError(err)
						continue
					}
					sourceContent, err := sourceClient.Stat(false)
					if err != nil {
						// source doesn't exist anymore
						mj.statusCh <- mirrorURL.WithError(err)
						continue
					}
					targetClient, err := newClient(targetPath)
					if err != nil {
						// cannot create targetclient
						mj.statusCh <- mirrorURL.WithError(err)
						return
					}
					shouldQueue := false
					if !mj.isForce {
						_, err = targetClient.Stat(false)
						if err == nil {
							continue
						} // doesn't exist
						shouldQueue = true
					}
					if shouldQueue || mj.isForce {
						mirrorURL.TotalCount = mj.TotalObjects
						mirrorURL.TotalSize = mj.TotalBytes
						// adjust total, because we want to show progress of the item still queued to be copied.
						mj.status.SetTotal(mj.status.Total() + sourceContent.Size).Update()
						mj.statusCh <- mj.doMirror(mirrorURL)
					}
					continue
				}
				shouldQueue := false
				if !mj.isForce {
					targetClient, err := newClient(targetPath)
					if err != nil {
						// cannot create targetclient
						mj.statusCh <- mirrorURL.WithError(err)
						return
					}
					_, err = targetClient.Stat(false)
					if err == nil {
						continue
					} // doesn't exist
					shouldQueue = true
				}
				if shouldQueue || mj.isForce {
					mirrorURL.SourceContent.Size = event.Size
					mirrorURL.TotalCount = mj.TotalObjects
					mirrorURL.TotalSize = mj.TotalBytes
					// adjust total, because we want to show progress of the itemj stiil queued to be copied.
					mj.status.SetTotal(mj.status.Total() + event.Size).Update()
					mj.statusCh <- mj.doMirror(mirrorURL)
				}
			} else if event.Type == EventRemove {
				mirrorURL := URLs{
					SourceAlias:   sourceAlias,
					SourceContent: nil,
					TargetAlias:   targetAlias,
					TargetContent: &clientContent{URL: *targetURL},
				}
				mirrorURL.TotalCount = mj.TotalObjects
				mirrorURL.TotalSize = mj.TotalBytes
				if mirrorURL.TargetContent != nil && mj.isRemove && mj.isForce {
					mj.statusCh <- mj.doRemove(mirrorURL)
				}
			}

		case err := <-mj.watcher.Errors():
			mj.watchErr = err
			switch err.ToGoError().(type) {
			case APINotImplemented:
				// Ignore error if API is not implemented.
				mj.watcherRunning = false
				return
			default:
				errorIf(err, "Unexpected error during monitoring.")
			}
		}
	}
}

func (mj *mirrorJob) watchSourceURL(recursive bool) *probe.Error {
	sourceClient, err := newClient(mj.sourceURL)
	if err == nil {
		return mj.watcher.Join(sourceClient, recursive)
	} // Failed to initialize client.
	return err
}

func (mj *mirrorJob) watchURL(sourceClient Client) *probe.Error {
	return mj.watcher.Join(sourceClient, true)
}

// Fetch urls that need to be mirrored
func (mj *mirrorJob) startMirror() {
	var totalBytes int64
	var totalObjects int64

	URLsCh := prepareMirrorURLs(mj.sourceURL, mj.targetURL, mj.isForce, mj.isFake, mj.isRemove, mj.excludeOptions)
	for {
		select {
		case sURLs, ok := <-URLsCh:
			if !ok {
				// finished harvesting urls
				return
			}
			if sURLs.Error != nil {
				if strings.Contains(sURLs.Error.ToGoError().Error(), " is a folder.") {
					mj.status.errorIf(sURLs.Error.Trace(), "Folder cannot be copied. Please use `...` suffix.")
				} else {
					mj.status.errorIf(sURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}
				continue
			}
			if sURLs.SourceContent != nil {
				// copy
				totalBytes += sURLs.SourceContent.Size
			}

			totalObjects++
			mj.TotalBytes = totalBytes
			mj.TotalObjects = totalObjects
			mj.status.SetTotal(totalBytes)

			// Save total count.
			sURLs.TotalCount = mj.TotalObjects
			// Save totalSize.
			sURLs.TotalSize = mj.TotalBytes

			if sURLs.SourceContent != nil {
				mj.statusCh <- mj.doMirror(sURLs)
			} else if sURLs.TargetContent != nil && mj.isRemove && mj.isForce {
				mj.statusCh <- mj.doRemove(sURLs)
			}
		case <-mj.trapCh:
			os.Exit(0)
		}
	}
}

// when using a struct for copying, we could save a lot of passing of variables
func (mj *mirrorJob) mirror() {
	if globalQuiet || globalJSON {
	} else {
		// Enable progress bar reader only during default mode
		mj.scanBar = scanBarFactory()
	}

	// start the status go routine
	mj.startStatus()

	// Starts additional watcher thread for watching for new events.
	if mj.isWatch {
		go mj.watchMirror()
	}

	// Start mirroring.
	mj.startMirror()

	if mj.isWatch {
		<-mj.trapCh
	}

	// Wait until all progress bar updates are actually shown and quit.
	mj.stopStatus()
}

func newMirrorJob(srcURL, dstURL string, isFake, isRemove, isWatch, isForce bool, excludeOptions []string) *mirrorJob {
	// we'll define the status to use here,
	// do we want the quiet status? or the progressbar
	var status = NewProgressStatus()
	if globalQuiet {
		status = NewQuietStatus()
	} else if globalJSON {
		status = NewDummyStatus()
	}

	mj := mirrorJob{
		trapCh: signalTrap(os.Interrupt, syscall.SIGTERM, syscall.SIGKILL),
		m:      new(sync.Mutex),

		sourceURL: srcURL,
		targetURL: dstURL,

		isFake:         isFake,
		isRemove:       isRemove,
		isWatch:        isWatch,
		isForce:        isForce,
		excludeOptions: excludeOptions,

		status:         status,
		scanBar:        func(s string) {},
		statusCh:       make(chan URLs),
		wgStatus:       new(sync.WaitGroup),
		watcherRunning: true,
		watcher:        NewWatcher(UTCNow()),
	}

	return &mj
}

// copyBucketPolicies - copy policies from source to dest
func copyBucketPolicies(srcClt, dstClt Client, isForce bool) *probe.Error {
	rules, err := srcClt.GetAccessRules()
	if err != nil {
		return err
	}
	// Set found rules to target bucket if permitted
	for _, r := range rules {
		originalRule, err := dstClt.GetAccess()
		if err != nil {
			return err
		}
		// Set rule only if it doesn't exist in the target bucket
		// or force flag is activated
		if originalRule == "none" || isForce {
			err = dstClt.SetAccess(r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// runMirror - mirrors all buckets to another S3 server
func runMirror(srcURL, dstURL string, ctx *cli.Context) *probe.Error {

	// Create a new mirror job and execute it
	mj := newMirrorJob(srcURL, dstURL,
		ctx.Bool("fake"),
		ctx.Bool("remove"),
		ctx.Bool("watch"),
		ctx.Bool("force"),
		ctx.StringSlice("exclude"))

	srcClt, err := newClient(srcURL)
	fatalIf(err, "Unable to initialize `"+srcURL+"`")

	dstClt, err := newClient(dstURL)
	fatalIf(err, "Unable to initialize `"+srcURL+"`")

	if ctx.Bool("a") && (srcClt.GetURL().Type != objectStorage || dstClt.GetURL().Type != objectStorage) {
		fatalIf(errDummy(), "Synchronizing bucket policies is only possible when both source & target point to S3 servers")
	}

	mirrorAllBuckets := (srcClt.GetURL().Type == objectStorage && srcClt.GetURL().Path == "/") ||
		(dstClt.GetURL().Type == objectStorage && dstClt.GetURL().Path == "/")

	if mirrorAllBuckets {
		// Synchronize buckets using dirDifference function
		for d := range dirDifference(srcClt, dstClt, srcURL, dstURL) {
			if d.Error != nil {
				mj.status.fatalIf(d.Error, fmt.Sprintf("Failed to start monitoring."))
			}
			if d.Diff == differInSecond {
				// Ignore buckets that only exist in target instance
				continue
			}

			sourceSuffix := strings.TrimPrefix(d.FirstURL, srcClt.GetURL().String())

			newSrcURL := path.Join(srcURL, sourceSuffix)
			newTgtURL := path.Join(dstURL, sourceSuffix)

			newSrcClt, _ := newClient(newSrcURL)
			newDstClt, _ := newClient(newTgtURL)

			if d.Diff == differInFirst {
				// Bucket only exists in the source, create the same bucket in the destination
				err := newDstClt.MakeBucket(ctx.String("region"), false)
				if err != nil {
					mj.mirrorErr = err
					errorIf(err, "Cannot created bucket in `"+newTgtURL+"`")
					continue
				}
				// Copy policy rules from source to dest if flag is activated
				if ctx.Bool("a") {
					err := copyBucketPolicies(srcClt, dstClt, ctx.Bool("force"))
					if err != nil {
						mj.mirrorErr = err
						errorIf(err, "Cannot copy bucket policies to `"+newDstClt.GetURL().String()+"`")
					}
				}
			}

			if mj.isWatch {
				// monitor mode will watch the source folders for changes,
				// and queue them for copying.
				if err := mj.watchURL(newSrcClt); err != nil {
					mj.status.fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
				}
			}
		}
	}

	if !mirrorAllBuckets && mj.isWatch {
		// monitor mode will watch the source folders for changes,
		// and queue them for copying.
		if err := mj.watchURL(srcClt); err != nil {
			mj.status.fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
		}
	}

	// Start mirroring job
	mj.mirror()

	// Check for errors during mirroring or watching to return
	if mj.mirrorErr != nil {
		return mj.mirrorErr
	}
	if mj.watchErr != nil {
		return mj.watchErr
	}

	return nil
}

// Main entry point for mirror command.
func mainMirror(ctx *cli.Context) error {

	// check 'mirror' cli arguments.
	checkMirrorSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("Mirror", color.New(color.FgGreen, color.Bold))

	args := ctx.Args()

	srcURL := args[0]
	tgtURL := args[1]

	if err := runMirror(srcURL, tgtURL, ctx); err != nil {
		return exitStatus(globalErrorExitStatus)
	}

	return nil
}
