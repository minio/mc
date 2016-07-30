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

// todo(nl5887):
// quiet should be really quiet
// hide progress bar -> no-progress

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

// cp command flags.
var (
	cpFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of cp.",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "Copy recursively.",
		},
		cli.BoolFlag{
			Name:  "skip-copy",
			Usage: "Skip copy (useful for only monitor)",
		},
		cli.BoolFlag{
			Name:  "monitor, m",
			Usage: "Monitor and apply changes.",
		},
	}
)

// Copy command.
var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "Copy one or more objects to a target.",
	Action: mainCopy,
	Flags:  append(cpFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] SOURCE [SOURCE...] TARGET

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Copy a list of objects from local file system to Amazon S3 cloud storage.
      $ mc {{.Name}} Music/*.ogg s3/jukebox/

   2. Copy a folder recursively from Minio cloud storage to Amazon S3 cloud storage.
      $ mc {{.Name}} --recursive play/mybucket/burningman2011/ s3/mybucket/

   3. Copy multiple local folders recursively to Minio cloud storage.
      $ mc {{.Name}} --recursive backup/2014/ backup/2015/ play/archive/

   4. Copy a bucket recursively from aliased Amazon S3 cloud storage to local filesystem on Windows.
      $ mc {{.Name}} --recursive s3\documents\2014\ C:\Backups\2014

   5. Copy an object with name containing unicode characters to Amazon S3 cloud storage.
      $ mc {{.Name}} 本語 s3/andoria/

   6. Copy a local folder with space separated characters to Amazon S3 cloud storage.
      $ mc {{.Name}} --recursive 'workdir/documents/May 2014/' s3/miniocloud
`,
}

// copyMessage container for file copy messages
type copyMessage struct {
	Status string `json:"status"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// String colorized copy message
func (c copyMessage) String() string {
	return console.Colorize("Copy", fmt.Sprintf("‘%s’ -> ‘%s’", c.Source, c.Target))
}

// JSON jsonified copy message
func (c copyMessage) JSON() string {
	c.Status = "success"
	copyMessageBytes, e := json.Marshal(c)
	fatalIf(probe.NewError(e), "Failed to marshal copy message.")

	return string(copyMessageBytes)
}

// copyStatMessage container for copy accounting message
type copyStatMessage struct {
	Total       int64
	Transferred int64
	Speed       float64
}

const (
	// 5GiB.
	fiveGB = 5 * 1024 * 1024 * 1024
)

// copyStatMessage copy accounting message
func (c copyStatMessage) String() string {
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

// doCopy - Copy a singe file from source to destination
func (cs *copySession) doCopy(cpURLs copyURLs) copyURLs {
	// why?
	if cpURLs.Error != nil {
		cpURLs.Error = cpURLs.Error.Trace()
		return cpURLs
	}

	sourceAlias := cpURLs.SourceAlias
	sourceURL := cpURLs.SourceContent.URL
	targetAlias := cpURLs.TargetAlias
	targetURL := cpURLs.TargetContent.URL
	length := cpURLs.SourceContent.Size

	cs.pb.SetCaption(sourceURL.String() + ": ")

	var progress io.Reader = cs.pr

	if globalQuiet || globalJSON {
		sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
		targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))
		printMsg(copyMessage{
			Source: sourcePath,
			Target: targetPath,
		})
	}

	// If source size is <= 5GB and operation is across same server type try to use Copy.
	if length <= fiveGB && (sourceURL.Type == targetURL.Type) {
		// FS -> FS Copy includes alias in path.
		if sourceURL.Type == fileSystem {
			sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
			err := copySourceStreamFromAlias(targetAlias, targetURL.String(), sourcePath, length, progress)
			if err != nil {
				cpURLs.Error = err.Trace(sourceURL.String())
				return cpURLs
			}
		} else if sourceURL.Type == objectStorage {
			// If source/target are object storage their aliases must be the same.
			if sourceAlias == targetAlias {
				// Do not include alias inside path for ObjStore -> ObjStore.
				err := copySourceStreamFromAlias(targetAlias, targetURL.String(), sourceURL.Path, length, progress)
				if err != nil {
					cpURLs.Error = err.Trace(sourceURL.String())
					return cpURLs
				}
			} else {
				reader, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
				if err != nil {
					cpURLs.Error = err.Trace(sourceURL.String())
					return cpURLs
				}
				_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, progress)
				if err != nil {
					cpURLs.Error = err.Trace(targetURL.String())
					return cpURLs
				}
			}
		}
	} else {
		// Standard GET/PUT for size > 5GB.
		reader, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
		if err != nil {
			cpURLs.Error = err.Trace(sourceURL.String())
			return cpURLs
		}
		_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, progress)
		if err != nil {
			cpURLs.Error = err.Trace(targetURL.String())
			return cpURLs
		}
	}
	cpURLs.Error = nil // just for safety
	return cpURLs
}

type copySession struct {
	*sessionV7

	trapCh <-chan bool

	statusCh  chan copyURLs
	harvestCh chan copyURLs
	errorCh   chan *probe.Error

	watcher *watcher
	queue   *Queue

	// mutex for shutdown
	m *sync.Mutex

	wgStatus *sync.WaitGroup
	wgCopy   *sync.WaitGroup

	// TODO(nl5887): accounter and pb should use same interface
	// these functions can be optimised more by fixing the
	// print and erase issue.
	accountingReader *accounter
	pb               *progressBar
	pr               io.Reader
	scanBar          scanBarFunc
	eraseLine        func()

	sourceURLs []string
	targetURL  string
}

func newCopySession(session *sessionV7) *copySession {
	pb := newProgressBar(0)
	args := session.Header.CommandArgs

	cs := copySession{
		trapCh:    signalTrap(os.Interrupt, syscall.SIGTERM),
		sessionV7: session,

		statusCh:  make(chan copyURLs),
		harvestCh: make(chan copyURLs),
		errorCh:   make(chan *probe.Error),

		watcher: NewWatcher(),

		// we're using a queue instead of a channel, this allows us to gracefully
		// stop. if we're using a channel and want to trap a signal, the channel
		// the backlog of fs events will be send on a closed channel.
		queue: NewQueue(session),

		wgStatus: new(sync.WaitGroup),
		wgCopy:   new(sync.WaitGroup),
		m:        new(sync.Mutex),

		accountingReader: newAccounter(0),
		pb:               pb,
		pr:               pb,

		sourceURLs: args[:len(args)-1],
		targetURL:  args[len(args)-1], // Last one is target

		scanBar: discardScanBarFactory,
		eraseLine: func() {
		},
	}

	return &cs
}

// Go routine to update session status
func (cs *copySession) startStatus() {
	cs.wgStatus.Add(1)

	go func() {
		defer cs.wgStatus.Done()

		for cpURLs := range cs.statusCh {
			if cpURLs.Error != nil {
				// Print in new line and adjust to top so that we
				// don't print over the ongoing progress bar.
				errorIf(cpURLs.Error.Trace(cpURLs.SourceContent.URL.String()),
					fmt.Sprintf("Failed to copy ‘%s’.", cpURLs.SourceContent.URL.String()))

				// For all non critical errors we can continue for the
				// remaining files.
				switch cpURLs.Error.ToGoError().(type) {
				// Handle this specifically for filesystem related errors.
				case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
					continue
				// Handle these specifically for object storage related errors.
				case BucketNameEmpty, ObjectMissing, ObjectAlreadyExists, ObjectAlreadyExistsAsDirectory, BucketDoesNotExist, BucketInvalid, ObjectOnGlacier:
					continue
				}

				// For critical errors we should exit. Session
				// can be resumed after the user figures out
				// the  problem. We know that
				// there are no current copy actions, because
				// we're not multi threading now.

				// this issue could be separated using separate
				// error channel instead of using cpURLs.Error
				cs.CloseAndDie()
			}

			// finished harvesting urls, save queue to session data
			if err := cs.queue.Save(cs.NewDataWriter()); err != nil {
				fatalIf(probe.NewError(err), "Unable to save queue.")
			}

			cs.Header.LastCopied = cpURLs.SourceContent.URL.String()
			cs.Save()
		}
	}()
}

func (cs *copySession) startCopy(wait bool) {
	cs.wgCopy.Add(1)

	go func() {
		defer cs.wgCopy.Done()

		for {
			if !wait {
			} else if err := cs.queue.Wait(); err != nil {
				break
			}

			cpURLs := cs.queue.Pop()
			if cpURLs == nil {
				break
			}

			cs.statusCh <- cs.doCopy(*(cpURLs).(*copyURLs))
		}
	}()
}

// this goroutine will watch for notifications, and add modified objects to the queue
func (cs *copySession) watch() {
	for {
		select {
		case event, ok := <-cs.watcher.Events():
			if !ok {
				// channel closed
				return
			}

			targetURL := cs.targetURL
			if rel, err := filepath.Rel(event.Client.GetURL().Path, event.Path); err == nil {
				targetURL = filepath.Join(targetURL, filepath.Dir(rel))
			}

			cpURLs := prepareCopyURLsTypeB(event.Path, targetURL)
			if cpURLs.Error != nil {
				cs.statusCh <- cpURLs
				continue
			}

			if err := cs.queue.Push(&cpURLs); err != nil {
				// will throw an error if already queue, ignoring this error
				continue
			}

			// adjust total, because we want to show progress of the items stiil
			// queued to be copied.
			cs.pb.SetTotal(cs.pb.Total + cpURLs.SourceContent.Size).Update()
		case err := <-cs.watcher.Errors():
			errorIf(err,
				fmt.Sprintf("Got error during monitoring."))
		}
	}
}

func (cs *copySession) watchSourceURLs(recursive bool) *probe.Error {
	for _, url := range cs.sourceURLs {
		if sourceClient, err := newClient(url); err != nil {
			return err
		} else if err := cs.watcher.Join(sourceClient, recursive /*, events to monitor here*/); err != nil {
			return err
		} else {
			// no errors, all set.
		}
	}

	return nil
}
func (cs *copySession) harvestSessionUrls() {
	defer close(cs.harvestCh)

	urlScanner := bufio.NewScanner(cs.NewDataReader())
	for urlScanner.Scan() {
		var cpURLs copyURLs

		if err := json.Unmarshal([]byte(urlScanner.Text()), &cpURLs); err != nil {
			cs.errorCh <- probe.NewError(err)
			continue
		}

		cs.harvestCh <- cpURLs
	}

	if err := urlScanner.Err(); err != nil {
		cs.errorCh <- probe.NewError(err)
	}
}

func (cs *copySession) harvestSourceUrls(recursive bool) {
	defer close(cs.harvestCh)

	URLsCh := prepareCopyURLs(cs.sourceURLs, cs.targetURL, recursive)

	for cpURLs := range URLsCh {
		cs.harvestCh <- cpURLs
	}
}

func (cs *copySession) harvest(recursive bool) {
	if cs.HasData() {
		// resume previous session, harvest urls from session
		go cs.harvestSessionUrls()
	} else {
		// harvest urls from source urls
		go cs.harvestSourceUrls(recursive)
	}

	var totalBytes int64
	var totalObjects int

loop:
	for {
		select {
		case cpURLs, ok := <-cs.harvestCh:
			if !ok {
				// finished harvesting urls
				break loop
			}

			if cpURLs.Error != nil {
				cs.eraseLine()

				if strings.Contains(cpURLs.Error.ToGoError().Error(), " is a folder.") {
					errorIf(cpURLs.Error.Trace(), "Folder cannot be copied. Please use ‘...’ suffix.")
				} else {
					errorIf(cpURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}

				continue
			}

			cs.scanBar(cpURLs.SourceContent.URL.String())

			cs.queue.Push(&cpURLs)

			totalBytes += cpURLs.SourceContent.Size
			totalObjects++
		case err := <-cs.errorCh:
			cs.eraseLine()
			fatalIf(err, "Unable to harvest URL for copying.")
		case <-cs.trapCh:
			cs.eraseLine()
			console.Println(console.Colorize("Copy", "Abort"))
			return
		}
	}

	// finished harvesting urls, save queue to session data
	if err := cs.queue.Save(cs.NewDataWriter()); err != nil {
		fatalIf(probe.NewError(err), "Unable to save queue.")
	}

	// update session file and save
	cs.Header.TotalBytes = totalBytes
	cs.Header.TotalObjects = totalObjects
	cs.Save()

	// update progressbar and accounting reader
	cs.pb.Total = totalBytes
	cs.accountingReader.Total = totalBytes
}

// when using a struct for copying, we could save a lot of passing of variables
func (cs *copySession) copy() {
	monitor := cs.Header.CommandBoolFlags["monitor"]
	skipCopy := cs.Header.CommandBoolFlags["skip-copy"]
	recursive := cs.Header.CommandBoolFlags["recursive"]

	// todo(nl5887): we can do this better
	if globalQuiet {
		cs.pb.Output = ioutil.Discard
	} else if globalJSON {
		cs.pb.Output = ioutil.Discard
	} else {
		// Enable progress bar reader only during default mode
		cs.scanBar = scanBarFactory()
		cs.eraseLine = console.Eraseline
	}

	if skipCopy {
		// we want to skip copy, so don't harvest urls.
	} else {
		cs.harvest(recursive)
	}

	// global quiet shows stats once and a while, I think quiet should be really
	// quiet, and have something like no-progress
	if globalQuiet {
		defer func() {
			accntStat := cs.accountingReader.Stat()
			cpStatMessage := copyStatMessage{
				Total:       accntStat.Total,
				Transferred: accntStat.Transferred,
				Speed:       accntStat.Speed,
			}
			console.Println(console.Colorize("Copy", cpStatMessage.String()))
		}()

		cs.pr = cs.accountingReader
	}

	// now we want to start the progress bar
	cs.pb.Start()
	defer cs.pb.Finish()

	cs.startStatus()

	go func() {
		// on SIGTERM shutdown and stop
		<-cs.trapCh

		cs.shutdown()
		cs.CloseAndDie()
	}()

	// monitor mode will watch the source folders for changes,
	// and queue them for copying. Monitor mode can be stopped
	// only by SIGTERM.
	if monitor {
		if err := cs.watchSourceURLs(recursive); err != nil {
			fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
		}

		cs.startCopy(true)

		cs.watch()

		// don't let monitor finish, only on SIGTERM
		done := make(chan bool)
		<-done
	} else {
		cs.startCopy(false)

		// wait for copy to finish
		cs.wgCopy.Wait()
		cs.shutdown()
	}
}

func (cs *copySession) shutdown() {
	// make sure only one shutdown can be active
	cs.m.Lock()

	cs.watcher.Stop()

	// stop queue, prevent new copy actions
	cs.queue.Close()

	// wait for current copy action to finish
	cs.wgCopy.Wait()

	close(cs.statusCh)

	// copy sends status message, wait for status
	cs.wgStatus.Wait()
}

// mainCopy is the entry point for cp command.
func mainCopy(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'copy' cli arguments.
	checkCopySyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Copy", color.New(color.FgGreen, color.Bold))

	session := newSessionV7()
	session.Header.CommandType = "cp"
	session.Header.CommandBoolFlags["recursive"] = ctx.Bool("recursive")
	session.Header.CommandBoolFlags["monitor"] = ctx.Bool("monitor")
	session.Header.CommandBoolFlags["skip-copy"] = ctx.Bool("skip-copy")

	var e error
	if session.Header.RootPath, e = os.Getwd(); e != nil {
		session.Delete()
		fatalIf(probe.NewError(e), "Unable to get current working folder.")
	}

	// extract URLs.
	session.Header.CommandArgs = ctx.Args()

	cs := newCopySession(session)

	// delete will be run when terminating normally,
	// on SIGTERM it won't execute
	defer cs.Delete()

	cs.copy()
}
