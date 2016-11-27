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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

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
			Name:  "help, h",
			Usage: "Show this help.",
		},
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
	}
)

//  Mirror folders recursively from a single source to many destinations
var mirrorCmd = cli.Command{
	Name:   "mirror",
	Usage:  "Mirror buckets and folders.",
	Action: mainMirror,
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

   6. Continuously mirror a local folder recursively to Minio cloud storage. '--watch' continuously watches for
      new objects and uploads them.
      $ mc {{.Name}} --force --remove --watch /var/lib/backups play/backups

`,
}

type mirrorSession struct {
	// embeds the session struct
	*sessionV8

	// the channel to trap SIGKILL signals
	trapCh <-chan bool

	// channel for status messages
	statusCh chan URLs
	// channel for urls to harvest
	harvestCh chan URLs
	// channel for errors
	errorCh chan *probe.Error

	// the global watcher object, which receives notifications of created
	// and deleted files
	watcher *Watcher

	// the queue of objects to be created or removed
	queue *Queue

	// mutex for shutdown, this prevents the shutdown
	// to be initiated multiple times
	m *sync.Mutex

	// waitgroup for status goroutine, waits till all status
	// messages have been written and received
	wgStatus *sync.WaitGroup

	// waitgroup for mirror goroutine, waits till all
	// mirror actions have been completed
	wgMirror *sync.WaitGroup

	status  Status
	scanBar scanBarFunc

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
func (ms *mirrorSession) doRemove(sURLs URLs) URLs {
	isFake := ms.Header.CommandBoolFlags["fake"]
	if isFake {
		return sURLs.WithError(nil)
	}

	targetURL := sURLs.TargetContent.URL

	// We are not removing incomplete uploads.
	isIncomplete := false

	// Remove extraneous file on target.
	removeSingle(targetURL.String(), isIncomplete, isFake, 0)

	return sURLs.WithError(nil)
}

// doMirror - Mirror an object to multiple destination. URLs status contains a copy of sURLs and error if any.
func (ms *mirrorSession) doMirror(sURLs URLs) URLs {
	isFake := ms.Header.CommandBoolFlags["fake"]

	if sURLs.Error != nil { // Errorneous sURLs passed.
		return sURLs.WithError(sURLs.Error.Trace())
	}

	//s For a fake mirror make sure we update respective progress bars
	// and accounting readers under relevant conditions.
	if isFake {
		ms.status.Add(sURLs.SourceContent.Size)
		return sURLs.WithError(nil)
	}

	sourceAlias := sURLs.SourceAlias
	sourceURL := sURLs.SourceContent.URL
	targetAlias := sURLs.TargetAlias
	targetURL := sURLs.TargetContent.URL
	length := sURLs.SourceContent.Size

	ms.status.SetCaption(sourceURL.String() + ": ")

	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))
	ms.status.PrintMsg(mirrorMessage{
		Source:     sourcePath,
		Target:     targetPath,
		Size:       length,
		TotalCount: sURLs.TotalCount,
		TotalSize:  sURLs.TotalSize,
	})

	// If source size is <= 5GB and operation is across same server type try to use Copy.
	if length <= fiveGB && sourceURL.Type == targetURL.Type {
		// FS -> FS Copy includes alias in path.
		if sourceURL.Type == fileSystem {
			sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
			err := copySourceStreamFromAlias(targetAlias, targetURL.String(), sourcePath, length, ms.status)
			if err != nil {
				return sURLs.WithError(err.Trace(sourceURL.String()))
			}
		} else if sourceURL.Type == objectStorage {
			if sourceAlias == targetAlias {
				// If source/target are object storage their aliases must be the same
				// Do not include alias inside path for ObjStore -> ObjStore.
				sourcePath := filepath.ToSlash(sourceURL.Path)
				err := copySourceStreamFromAlias(targetAlias, targetURL.String(), sourcePath, length, ms.status)
				if err != nil {
					return sURLs.WithError(err.Trace(sourceURL.String()))
				}
			} else {
				reader, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
				if err != nil {
					return sURLs.WithError(err.Trace(sourceURL.String()))
				}
				_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, ms.status)
				if err != nil {
					return sURLs.WithError(err.Trace(targetURL.String()))
				}
			}
		}
	} else {
		// Standard GET/PUT for size > 5GB
		reader, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
		if err != nil {
			return sURLs.WithError(err.Trace(sourceURL.String()))
		}
		_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, ms.status)
		if err != nil {
			return sURLs.WithError(err.Trace(targetURL.String()))
		}
	}

	return sURLs.WithError(nil)
}

// Go routine to update session status
func (ms *mirrorSession) startStatus() {
	ms.wgStatus.Add(1)

	// wait for status messages on statusChan, show error messages and write the current queue to session
	go func() {
		defer ms.wgStatus.Done()

		for sURLs := range ms.statusCh {
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
				if ms.Header.CommandBoolFlags["watch"] {
					continue
				}
				// For all non critical errors we can continue for the remaining files.
				if isErrIgnored(sURLs.Error) {
					continue
				}

				// For critical errors we should exit. Session
				// can be resumed after the user figures out
				// the  problem. We know that
				// there are no current copy actions, because
				// we're not multi threading now.

				// this issue could be separated using separate
				// error channel instead of using sURLs.Error
				ms.CloseAndDie()
			}

			// finished harvesting urls, save queue to session data
			if err := ms.queue.Save(ms.NewDataWriter()); err != nil {
				ms.status.fatalIf(probe.NewError(err), "Unable to save queue.")
			}

			if sURLs.SourceContent != nil {
				ms.Header.LastCopied = sURLs.SourceContent.URL.String()
			} else if sURLs.TargetContent != nil {
				ms.Header.LastRemoved = sURLs.TargetContent.URL.String()
			}

			ms.Save()

			if sURLs.SourceContent != nil {
			} else if sURLs.TargetContent != nil {
				// Construct user facing message and path.
				targetPath := filepath.ToSlash(filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path))
				ms.status.PrintMsg(rmMessage{Key: targetPath})
			}
		}
	}()
}

func (ms *mirrorSession) startMirror(wait bool) {
	isRemove := ms.Header.CommandBoolFlags["remove"]

	ms.wgMirror.Add(1)

	// wait for new urls to mirror or delete in the queue, and
	// run the actual mirror or remove.
	go func() {
		defer ms.wgMirror.Done()

		for {
			if !wait {
			} else if err := ms.queue.Wait(); err != nil {
				break
			}

			v := ms.queue.Pop()
			if v == nil {
				break
			}

			sURLs, ok := v.(URLs)
			if !ok {
				fatalIf(errInvalidArgument(), fmt.Sprintf("URLs type not found, %#v", v))
			}

			// Save total count.
			sURLs.TotalCount = ms.Header.TotalObjects

			// Save totalSize.
			sURLs.TotalSize = ms.Header.TotalBytes

			if sURLs.SourceContent != nil {
				ms.statusCh <- ms.doMirror(sURLs)
			} else if sURLs.TargetContent != nil && isRemove {
				ms.statusCh <- ms.doRemove(sURLs)
			}
		}
	}()
}

// this goroutine will watch for notifications, and add modified objects to the queue
func (ms *mirrorSession) watch() {
	isForce := ms.Header.CommandBoolFlags["force"]

	for {
		select {
		case event, ok := <-ms.watcher.Events():
			if !ok {
				// channel closed
				return
			}

			// this code seems complicated, it will change the expanded alias back to the alias
			// again, by replacing the sourceUrlFull with the sourceAlias. This url will be
			// used to mirror.
			sourceAlias, sourceURLFull, _ := mustExpandAlias(ms.sourceURL)

			// If the passed source URL points to fs, fetch the absolute src path
			// to correctly calculate targetPath
			if sourceAlias == "" {
				tmpSrcURL, err := filepath.Abs(sourceURLFull)
				if err == nil {
					sourceURLFull = tmpSrcURL
				}
			}

			sourceURL := newClientURL(event.Path)

			aliasedPath := strings.Replace(event.Path, sourceURLFull, ms.sourceURL, -1)

			// build target path, it is the relative of the event.Path with the sourceUrl
			// joined to the targetURL.
			sourceSuffix := strings.TrimPrefix(event.Path, sourceURLFull)
			targetPath := urlJoinPath(ms.targetURL, sourceSuffix)

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
						ms.statusCh <- mirrorURL.WithError(err)
						continue
					}
					sourceContent, err := sourceClient.Stat(false)
					if err != nil {
						// source doesn't exist anymore
						ms.statusCh <- mirrorURL.WithError(err)
						continue
					}
					targetClient, err := newClient(targetPath)
					if err != nil {
						// cannot create targetclient
						ms.statusCh <- mirrorURL.WithError(err)
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
						mirrorURL.TotalCount = ms.Header.TotalObjects
						mirrorURL.TotalSize = ms.Header.TotalBytes
						if e := ms.queue.Push(mirrorURL); e != nil {
							// will throw an error if already queue, ignoring this error
							continue
						}
						// adjust total, because we want to show progress of the items stiil
						// queued to be copied.
						ms.status.SetTotal(ms.status.Total() + sourceContent.Size).Update()
					}
					continue
				}
				shouldQueue := false
				if !isForce {
					targetClient, err := newClient(targetPath)
					if err != nil {
						// cannot create targetclient
						ms.statusCh <- mirrorURL.WithError(err)
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
					mirrorURL.TotalCount = ms.Header.TotalObjects
					mirrorURL.TotalSize = ms.Header.TotalBytes
					if e := ms.queue.Push(mirrorURL); e != nil {
						// will throw an error if already queue, ignoring this error
						continue
					}
					// adjust total, because we want to show progress of the items stiil queued to be copied.
					ms.status.SetTotal(ms.status.Total() + event.Size).Update()
				}
			} else if event.Type == EventRemove {
				mirrorURL := URLs{
					SourceAlias:   sourceAlias,
					SourceContent: nil,
					TargetAlias:   targetAlias,
					TargetContent: &clientContent{URL: *targetURL},
				}
				mirrorURL.TotalCount = ms.Header.TotalObjects
				mirrorURL.TotalSize = ms.Header.TotalBytes
				if err := ms.queue.Push(mirrorURL); err != nil {
					// will throw an error if already queue, ignoring this error
					continue
				}
			}

		case err := <-ms.watcher.Errors():
			errorIf(err, "Unexpected error during monitoring.")
		}
	}
}

func (ms *mirrorSession) watchSourceURL(recursive bool) *probe.Error {
	sourceClient, err := newClient(ms.sourceURL)
	if err == nil {
		return ms.watcher.Join(sourceClient, recursive)
	} // Failed to initialize client.
	return err
}

func (ms *mirrorSession) harvestSessionUrls() {
	defer close(ms.harvestCh)

	urlScanner := bufio.NewScanner(ms.NewDataReader())
	for urlScanner.Scan() {
		var urls URLs

		if err := json.Unmarshal([]byte(urlScanner.Text()), &urls); err != nil {
			ms.errorCh <- probe.NewError(err)
			continue
		}

		// Send harvested urls.
		ms.harvestCh <- urls
	}

	if err := urlScanner.Err(); err != nil {
		ms.errorCh <- probe.NewError(err)
	}
}

func (ms *mirrorSession) harvestSourceUrls(recursive bool) {
	isForce := ms.Header.CommandBoolFlags["force"]
	isFake := ms.Header.CommandBoolFlags["fake"]
	isRemove := ms.Header.CommandBoolFlags["remove"]
	isWatch := ms.Header.CommandBoolFlags["watch"]

	defer close(ms.harvestCh)

	URLsCh := prepareMirrorURLs(ms.sourceURL, ms.targetURL, isForce, isFake, isRemove, isWatch)
	for url := range URLsCh {
		// Send harvested urls.
		ms.harvestCh <- url
	}
}

func (ms *mirrorSession) harvest(recursive bool) {
	isRemove := ms.Header.CommandBoolFlags["remove"]

	if ms.HasData() {
		// resume previous session, harvest urls from session
		go ms.harvestSessionUrls()
	} else {
		// harvest urls from source urls
		go ms.harvestSourceUrls(recursive)
	}

	var totalBytes int64
	var totalObjects int64

loop:
	for {
		select {
		case sURLs, ok := <-ms.harvestCh:
			if !ok {
				// finished harvesting urls
				break loop
			}
			if sURLs.Error != nil {
				if strings.Contains(sURLs.Error.ToGoError().Error(), " is a folder.") {
					ms.status.errorIf(sURLs.Error.Trace(), "Folder cannot be copied. Please use ‘...’ suffix.")
				} else {
					ms.status.errorIf(sURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}
				continue
			}

			if sURLs.SourceContent != nil {
				// copy
				ms.scanBar(sURLs.SourceContent.URL.String())
				totalBytes += sURLs.SourceContent.Size
			} else if sURLs.TargetContent != nil && isRemove {
				// delete
				ms.scanBar(sURLs.TargetContent.URL.String())
			}

			ms.queue.Push(sURLs)

			totalObjects++
		case err := <-ms.errorCh:
			ms.status.errorIf(err, "Unable to harvest URL for copying.")
			continue
		case <-ms.trapCh:
			ms.status.Println(console.Colorize("Mirror", "Abort"))
			return
		}
	}

	// finished harvesting urls, save queue to session data
	if err := ms.queue.Save(ms.NewDataWriter()); err != nil {
		ms.status.fatalIf(probe.NewError(err), "Unable to save queue.")
	}

	// update session file and save
	ms.Header.TotalBytes = totalBytes
	ms.Header.TotalObjects = totalObjects
	ms.Save()

	// update progressbar and accounting reader
	ms.status.SetTotal(totalBytes)
}

// when using a struct for copying, we could save a lot of passing of variables
func (ms *mirrorSession) mirror() {
	watch := ms.Header.CommandBoolFlags["watch"]
	recursive := ms.Header.CommandBoolFlags["recursive"]

	if globalQuiet {
	} else if globalJSON {
	} else {
		// Enable progress bar reader only during default mode
		ms.scanBar = scanBarFactory()
	}

	// harvest urls to copy
	ms.harvest(recursive)

	// now we want to start the progress bar
	ms.status.Start()
	defer ms.status.Finish()

	// start the status go routine
	ms.startStatus()

	// wait for trap signal to close properly and show message if there are
	// items queued left to resume the session.
	go func() {
		// on SIGTERM shutdown and stop
		<-ms.trapCh

		// Shutdown gracefully.
		ms.shutdown()

		// Save session and die.
		ms.CloseAndDie()
	}()

	// monitor mode will watch the source folders for changes,
	// and queue them for copying. Monitor mode can be stopped
	// only by SIGTERM.
	if watch {
		if err := ms.watchSourceURL(true); err != nil {
			ms.status.fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
		}

		ms.startMirror(true)

		ms.watch()
	} else {
		ms.startMirror(false)

		// wait for copy to finish
		ms.wgMirror.Wait()
		ms.shutdown()
	}
}

// Called upon signal trigger.
func (ms *mirrorSession) shutdown() {
	// make sure only one shutdown can be active
	ms.m.Lock()

	ms.watcher.Stop()

	// stop queue, prevent new copy actions
	ms.queue.Close()

	// wait for current copy action to finish
	ms.wgMirror.Wait()

	close(ms.statusCh)

	// copy sends status message, wait for status
	ms.wgStatus.Wait()
}

func newMirrorSession(session *sessionV8) *mirrorSession {
	args := session.Header.CommandArgs

	// we'll define the status to use here,
	// do we want the quiet status? or the progressbar
	var status = NewProgressStatus()
	if globalQuiet {
		status = NewQuietStatus()
	} else if globalJSON {
		status = NewDummyStatus()
	}

	ms := mirrorSession{
		trapCh:    signalTrap(os.Interrupt, syscall.SIGTERM),
		sessionV8: session,

		statusCh:  make(chan URLs),
		harvestCh: make(chan URLs),
		errorCh:   make(chan *probe.Error),

		watcher: NewWatcher(session.Header.When),

		// we're using a queue instead of a channel, this allows us to gracefully
		// stop. if we're using a channel and want to trap a signal, the channel
		// the backlog of fs events will be send on a closed channel.
		queue: NewQueue(),

		wgStatus: new(sync.WaitGroup),
		wgMirror: new(sync.WaitGroup),
		m:        new(sync.Mutex),

		status: status,
		// scanbar starts with no action
		scanBar: func(s string) {
		},

		sourceURL: args[0],
		targetURL: args[len(args)-1], // Last one is target
	}

	return &ms
}

// Main entry point for mirror command.
func mainMirror(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'mirror' cli arguments.
	checkMirrorSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Mirror", color.New(color.FgGreen, color.Bold))

	session := newSessionV8()
	session.Header.CommandType = "mirror"

	if v, err := os.Getwd(); err == nil {
		session.Header.RootPath = v
	} else {
		session.Delete()
		fatalIf(probe.NewError(err), "Unable to get current working folder.")
	}

	// Set command flags from context.
	session.Header.CommandBoolFlags["force"] = ctx.Bool("force")
	session.Header.CommandBoolFlags["fake"] = ctx.Bool("fake")
	session.Header.CommandBoolFlags["watch"] = ctx.Bool("watch")
	session.Header.CommandBoolFlags["remove"] = ctx.Bool("remove")

	// extract URLs.
	session.Header.CommandArgs = ctx.Args()

	ms := newMirrorSession(session)

	// Mirroring.
	ms.mirror()

	// delete will be run when terminating normally,
	ms.Delete()
}
