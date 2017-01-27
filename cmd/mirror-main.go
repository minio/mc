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
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
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
			Name:  "remove",
			Usage: "Remove extraneous file(s) on target.",
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
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] SOURCE TARGET

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Mirror a bucket recursively from Minio cloud storage to a bucket on Amazon S3 cloud storage.
      $ mc {{.Name}} play/photos/2014 s3/backup-photos

   2. Mirror a local folder recursively to Amazon S3 cloud storage.
      $ mc {{.Name}} backup/ s3/archive

   3. Mirror a bucket from aliased Amazon S3 cloud storage to a folder on Windows.
      $ mc {{.Name}} s3\documents\2014\ C:\backup\2014

   4. Mirror a bucket from aliased Amazon S3 cloud storage to a local folder use '--force' to overwrite destination.
      $ mc {{.Name}} --force s3/miniocloud miniocloud-backup

   5. Mirror a bucket from Minio cloud storage to a bucket on Amazon S3 cloud storage and remove any extraneous
      files on Amazon S3 cloud storage. NOTE: '--remove' is only supported with '--force'.
      $ mc {{.Name}} --force --remove play/photos/2014 s3/backup-photos/2014

`,
}

type mirrorJob struct {

	// the channel to trap SIGKILL signals
	trapCh <-chan bool

	// Hold context information
	context *cli.Context

	// mutex for shutdown, this prevents the shutdown
	// to be initiated multiple times
	m *sync.Mutex

	// channel for errors
	errorCh chan *probe.Error

	// the global watcher object, which receives notifications of created
	// and deleted files
	watcher *Watcher

	// the queue of objects to be created or removed
	mirrorQueue chan URLs
	// waitgroup for mirror goroutine, waits till all
	// mirror actions have been completed
	wgMirror *sync.WaitGroup

	// waitgroup for harvest goroutine
	wgHarvest *sync.WaitGroup
	// channel to halt harvest routine
	harvestStop chan bool
	// channel for urls to harvest
	harvestCh chan URLs

	// Hold operation status information
	status  Status
	scanBar scanBarFunc
	// waitgroup for status goroutine, waits till all status
	// messages have been written and received
	wgStatus *sync.WaitGroup
	// channel for status messages
	statusCh chan URLs

	TotalObjects int64
	TotalBytes   int64

	sourceURL string
	targetURL string
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
	return console.Colorize("Mirror", fmt.Sprintf("‘%s’ -> ‘%s’", m.Source, m.Target))
}

// JSON jsonified mirror message
func (m mirrorMessage) JSON() string {
	m.Status = "success"
	mirrorMessageBytes, e := json.Marshal(m)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(mirrorMessageBytes)
}

// mirrorStatMessage container for mirror accounting message
type mirrorStatMessage struct {
	Total       int64
	Transferred int64
	Speed       float64
}

// mirrorStatMessage mirror accounting message
func (c mirrorStatMessage) String() string {
	speedBox := pb.Format(int64(c.Speed)).To(pb.U_BYTES).String()
	if speedBox == "" {
		speedBox = "0 MB"
	} else {
		speedBox = speedBox + "/s"
	}
	message := fmt.Sprintf("Total: %s, Transferred: %s, Speed: %s", pb.Format(c.Total).To(pb.U_BYTES),
		pb.Format(c.Transferred).To(pb.U_BYTES), speedBox)
	return message
}

// doRemove - removes files on target.
func (mj *mirrorJob) doRemove(sURLs URLs) URLs {
	isFake := mj.context.Bool("fake")
	if isFake {
		return sURLs.WithError(nil)
	}

	// We are not removing incomplete uploads.
	isIncomplete := false

	// Construct proper path with alias.
	targetWithAlias := filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path)

	// Remove extraneous file on target.
	err := probe.NewError(removeSingle(targetWithAlias, isIncomplete, isFake, 0))
	return sURLs.WithError(err)
}

// doMirror - Mirror an object to multiple destination. URLs status contains a copy of sURLs and error if any.
func (mj *mirrorJob) doMirror(sURLs URLs) URLs {
	isFake := mj.context.Bool("fake")

	if sURLs.Error != nil { // Erroneous sURLs passed.
		return sURLs.WithError(sURLs.Error.Trace())
	}

	//s For a fake mirror make sure we update respective progress bars
	// and accounting readers under relevant conditions.
	if isFake {
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
		defer mj.status.Finish()
		defer mj.wgStatus.Done()

		for sURLs := range mj.statusCh {
			if sURLs.Error != nil {
				// Print in new line and adjust to top so that we
				// don't print over the ongoing progress bar.
				if sURLs.SourceContent != nil {
					errorIf(sURLs.Error.Trace(sURLs.SourceContent.URL.String()),
						fmt.Sprintf("Failed to copy ‘%s’.", sURLs.SourceContent.URL.String()))
				} else {
					// When sURLs.SourceContent is nil, we know that we have an error related to removing
					errorIf(sURLs.Error.Trace(sURLs.TargetContent.URL.String()),
						fmt.Sprintf("Failed to remove ‘%s’.", sURLs.TargetContent.URL.String()))
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

func (mj *mirrorJob) startMirror() {
	isRemove := mj.context.Bool("remove")
	mj.wgMirror.Add(1)
	// wait for new urls to mirror or delete in the queue, and
	// run the actual mirror or remove.
	defer mj.wgMirror.Done()

loop:
	for {
		select {
		case sURLs, ok := <-mj.mirrorQueue:
			if !ok {
				break loop
			}
			// Save total count.
			sURLs.TotalCount = mj.TotalObjects
			// Save totalSize.
			sURLs.TotalSize = mj.TotalBytes
			if sURLs.SourceContent != nil {
				mj.statusCh <- mj.doMirror(sURLs)
			} else if sURLs.TargetContent != nil && isRemove {
				mj.statusCh <- mj.doRemove(sURLs)
			}
		}
	}
}

// this goroutine will watch for notifications, and add modified objects to the queue
func (mj *mirrorJob) watch() {
	isForce := mj.context.Bool("force")

	for {
		select {
		case event, ok := <-mj.watcher.Events():
			if !ok {
				// channel closed
				return
			}
			// this code seemj complicated, it will change the expanded alias back to the alias
			// again, by replacing the sourceUrlFull with the sourceAlias. This url will be
			// used to mirror.
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

			// todo(nl5887): do we want all those actions here? those could cause the channels to
			// block in case of large num of changes
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
					if !isForce {
						_, err = targetClient.Stat(false)
						if err == nil {
							continue
						} // doesn't exist
						shouldQueue = true
					}
					if shouldQueue || isForce {
						mirrorURL.TotalCount = mj.TotalObjects
						mirrorURL.TotalSize = mj.TotalBytes
						mj.mirrorQueue <- mirrorURL
						// adjust total, because we want to show progress of the itemj stiil
						// queued to be copied.
						mj.status.SetTotal(mj.status.Total() + sourceContent.Size).Update()
					}
					continue
				}
				shouldQueue := false
				if !isForce {
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
				if shouldQueue || isForce {
					mirrorURL.SourceContent.Size = event.Size
					mirrorURL.TotalCount = mj.TotalObjects
					mirrorURL.TotalSize = mj.TotalBytes
					mj.mirrorQueue <- mirrorURL
					// adjust total, because we want to show progress of the itemj stiil queued to be copied.
					mj.status.SetTotal(mj.status.Total() + event.Size).Update()
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
				mj.mirrorQueue <- mirrorURL
			}

		case err := <-mj.watcher.Errors():
			errorIf(err, "Unexpected error during monitoring.")
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

func (mj *mirrorJob) harvestSourceUrls(recursive bool) {
	isForce := mj.context.Bool("force")
	isFake := mj.context.Bool("fake")
	isRemove := mj.context.Bool("remove")

	defer close(mj.harvestCh)

	URLsCh := prepareMirrorURLs(mj.sourceURL, mj.targetURL, isForce, isFake, isRemove)
	for {
		select {
		case <-mj.harvestStop:
			return
		case url := <-URLsCh:
			// Send harvested urls.
			mj.harvestCh <- url
		}
	}
}

// Fetch urls that need to be mirrored
func (mj *mirrorJob) harvest(recursive bool) {
	mj.wgHarvest.Add(1)
	defer mj.wgHarvest.Done()

	// harvest urls from source urls
	go mj.harvestSourceUrls(recursive)

	var totalBytes int64
	var totalObjects int64

loop:
	for {
		select {
		case sURLs, ok := <-mj.harvestCh:
			if !ok {
				// finished harvesting urls
				break loop
			}
			if sURLs.Error != nil {
				if strings.Contains(sURLs.Error.ToGoError().Error(), " is a folder.") {
					mj.status.errorIf(sURLs.Error.Trace(), "Folder cannot be copied. Please use ‘...’ suffix.")
				} else {
					mj.status.errorIf(sURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}
				continue
			}
			if sURLs.SourceContent != nil {
				// copy
				totalBytes += sURLs.SourceContent.Size
			}
			mj.mirrorQueue <- sURLs
			totalObjects++

			mj.status.SetTotal(totalBytes)

		case err := <-mj.errorCh:
			mj.status.errorIf(err, "Unable to harvest URL for copying.")
			continue
		}
	}

	mj.TotalBytes = totalBytes
	mj.TotalObjects = totalObjects
	// update progressbar and accounting reader
	mj.status.SetTotal(totalBytes)
}

// when using a struct for copying, we could save a lot of passing of variables
func (mj *mirrorJob) mirror() {
	recursive := mj.context.Bool("recursive")

	if globalQuiet || globalJSON {
	} else {
		// Enable progress bar reader only during default mode
		mj.scanBar = scanBarFactory()
	}

	// start the status go routine
	mj.startStatus()

	// harvest urls to copy
	go mj.harvest(recursive)

	// wait for trap signal to close properly
	go func() {
		// on SIGTERM shutdown and stop
		<-mj.trapCh

		// Shutdown gracefully.
		mj.shutdown()
	}()

	// monitor mode will watch the source folders for changes,
	// and queue them for copying. Monitor mode can be stopped
	// only by SIGTERM.
	if err := mj.watchSourceURL(true); err != nil {
		mj.status.fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
	}

	go mj.watch()

	mj.startMirror()
}

func (mj *mirrorJob) stopWatcher() {
	// Stop events watcher
	mj.watcher.Stop()
}

func (mj *mirrorJob) stopHarvester() {
	// Ask harvester to stop searching for new objects to mirror
	close(mj.harvestStop)
	mj.wgHarvest.Wait()
}

func (mj *mirrorJob) stopMirror() {
	// wait for current copy action to finish and stop copying
	close(mj.mirrorQueue)
	mj.wgMirror.Wait()
}

func (mj *mirrorJob) stopStatus() {
	// Gracefully stopping status
	close(mj.statusCh)
	mj.wgStatus.Wait()

}

// Called upon signal trigger.
func (mj *mirrorJob) shutdown() {
	// make sure only one shutdown can be active
	mj.m.Lock()

	// Stop everthing
	mj.stopWatcher()
	mj.stopHarvester()
	mj.stopMirror()
	mj.stopStatus()
}

func newMirrorJob(ctx *cli.Context) *mirrorJob {
	args := ctx.Args()

	// we'll define the status to use here,
	// do we want the quiet status? or the progressbar
	var status = NewProgressStatus()
	if globalQuiet {
		status = NewQuietStatus()
	} else if globalJSON {
		status = NewDummyStatus()
	}

	mj := mirrorJob{
		context: ctx,
		trapCh:  signalTrap(os.Interrupt, syscall.SIGTERM, syscall.SIGKILL),
		m:       new(sync.Mutex),

		sourceURL: args[0],
		targetURL: args[len(args)-1], // Last one is target

		errorCh: make(chan *probe.Error),

		harvestStop: make(chan bool),
		harvestCh:   make(chan URLs, 10000),
		wgHarvest:   new(sync.WaitGroup),

		status:   status,
		scanBar:  func(s string) {},
		statusCh: make(chan URLs),
		wgStatus: new(sync.WaitGroup),

		mirrorQueue: make(chan URLs, 1000),
		wgMirror:    new(sync.WaitGroup),

		watcher: NewWatcher(time.Now().UTC()),
	}

	return &mj
}

// Main entry point for mirror command.
func mainMirror(ctx *cli.Context) error {
	// check 'mirror' cli arguments.
	checkMirrorSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("Mirror", color.New(color.FgGreen, color.Bold))

	mj := newMirrorJob(ctx)
	mj.mirror()

	return nil
}
