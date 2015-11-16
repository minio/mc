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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

var (
	cpFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of cp",
	}
)

// Copy command.
var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "Copy one or more objects to a target.",
	Action: mainCopy,
	Flags:  []cli.Flag{cpFlagHelp},
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} SOURCE [SOURCE...] TARGET

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Copy a list of objects from local file system to Amazon S3 cloud storage.
      $ mc {{.Name}} Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio cloud storage to Amazon S3 cloud storage.
      $ mc {{.Name}} https://play.minio.io:9000/photobucket/burningman2011... https://s3.amazonaws.com/mybucket/

   3. Copy multiple local folders recursively to Minio cloud storage.
      $ mc {{.Name}} backup/2014/... backup/2015/... https://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 cloud storage to local filesystem on Windows.
      $ mc {{.Name}} s3/documents/2014/... C:\Backups\2014

   5. Copy an object with name containing unicode characters to Amazon S3 cloud storage.
      $ mc {{.Name}} 本語 s3/andoria/

   6. Copy a local folder with space separated characters to Amazon S3 cloud storage.
      $ mc {{.Name}} 'workdir/documents/May 2014...' s3/miniocloud
`,
}

// copyMessage container for file copy messages
type copyMessage struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Length int64  `json:"length"`
}

// String colorized copy message
func (c copyMessage) String() string {
	return console.Colorize("Copy", fmt.Sprintf("‘%s’ -> ‘%s’", c.Source, c.Target))
}

// JSON jsonified copy message
func (c copyMessage) JSON() string {
	copyMessageBytes, err := json.Marshal(c)
	fatalIf(probe.NewError(err), "Failed to marshal copy message.")

	return string(copyMessageBytes)
}

// doCopy - Copy a singe file from source to destination
func doCopy(cpURLs copyURLs, progressReader interface{}, cpQueue <-chan bool, wg *sync.WaitGroup, statusCh chan<- copyURLs) {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-cpQueue
	}()

	if cpURLs.Error != nil {
		cpURLs.Error.Trace()
		statusCh <- cpURLs
		return
	}

	if !globalQuietFlag && !globalJSONFlag {
		progressReader.(*barSend).SetCaption(cpURLs.SourceContent.URL.String() + ": ")
	}

	reader, length, err := getSource(cpURLs.SourceContent.URL.String())
	if err != nil {
		if !globalQuietFlag && !globalJSONFlag {
			progressReader.(*barSend).ErrorGet(length)
		}
		cpURLs.Error = err.Trace()
		statusCh <- cpURLs
		return
	}

	var newReader io.ReadCloser
	if globalQuietFlag || globalJSONFlag {
		printMsg(copyMessage{
			Source: cpURLs.SourceContent.URL.String(),
			Target: cpURLs.TargetContent.URL.String(),
			Length: cpURLs.SourceContent.Size,
		})
		newReader = progressReader.(*accounter).NewProxyReader(reader)
	} else {
		// set up progress
		newReader = progressReader.(*barSend).NewProxyReader(reader)
	}
	defer newReader.Close()

	if err := putTarget(cpURLs.TargetContent.URL.String(), length, newReader); err != nil {
		if !globalQuietFlag && !globalJSONFlag {
			progressReader.(*barSend).ErrorPut(length)
		}
		cpURLs.Error = err.Trace()
		statusCh <- cpURLs
		return
	}

	cpURLs.Error = nil // just for safety
	statusCh <- cpURLs
}

// doCopyFake - Perform a fake copy to update the progress bar appropriately.
func doCopyFake(cURLs copyURLs, progressReader interface{}) {
	if !globalQuietFlag && !globalJSONFlag {
		progressReader.(*barSend).Progress(cURLs.SourceContent.Size)
	}
}

// doPrepareCopyURLs scans the source URL and prepares a list of objects for copying.
func doPrepareCopyURLs(session *sessionV3, trapCh <-chan bool) {
	// Separate source and target. 'cp' can take only one target,
	// but any number of sources, even the recursive URLs mixed in-between.
	sourceURLs := session.Header.CommandArgs[:len(session.Header.CommandArgs)-1]
	targetURL := session.Header.CommandArgs[len(session.Header.CommandArgs)-1] // Last one is target

	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	var scanBar scanBarFunc
	if !globalQuietFlag && !globalJSONFlag { // set up progress bar
		scanBar = scanBarFactory()
	}

	URLsCh := prepareCopyURLs(sourceURLs, targetURL)
	done := false

	for done == false {
		select {
		case cpURLs, ok := <-URLsCh:
			if !ok { // Done with URL prepration
				done = true
				break
			}
			if cpURLs.Error != nil {
				// Print in new line and adjust to top so that we don't print over the ongoing scan bar
				if !globalQuietFlag && !globalJSONFlag {
					console.Eraseline()
				}
				if strings.Contains(cpURLs.Error.ToGoError().Error(), " is a folder.") {
					errorIf(cpURLs.Error.Trace(), "Folder cannot be copied. Please use ‘...’ suffix.")
				} else {
					errorIf(cpURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}
				break
			}

			jsonData, err := json.Marshal(cpURLs)
			if err != nil {
				session.Delete()
				fatalIf(probe.NewError(err), "Unable to prepare URL for copying. Error in JSON marshaling.")
			}
			fmt.Fprintln(dataFP, string(jsonData))
			if !globalQuietFlag && !globalJSONFlag {
				scanBar(cpURLs.SourceContent.URL.String())
			}

			totalBytes += cpURLs.SourceContent.Size
			totalObjects++
		case <-trapCh:
			// Print in new line and adjust to top so that we don't print over the ongoing scan bar
			if !globalQuietFlag && !globalJSONFlag {
				console.Eraseline()
			}
			session.Delete() // If we are interrupted during the URL scanning, we drop the session.
			os.Exit(0)
		}
	}
	session.Header.TotalBytes = totalBytes
	session.Header.TotalObjects = totalObjects
	session.Save()
}

func doCopySession(session *sessionV3) {
	trapCh := signalTrap(os.Interrupt, os.Kill)

	if !session.HasData() {
		doPrepareCopyURLs(session, trapCh)
	}

	var progressReader interface{}
	if !globalQuietFlag && !globalJSONFlag { // set up progress bar
		progressReader = newProgressBar(session.Header.TotalBytes)
	} else {
		progressReader = newAccounter(session.Header.TotalBytes)
	}

	// Prepare URL scanner from session data file.
	scanner := bufio.NewScanner(session.NewDataReader())
	// isCopied returns true if an object has been already copied
	// or not. This is useful when we resume from a session.
	isCopied := isCopiedFactory(session.Header.LastCopied)

	wg := new(sync.WaitGroup)
	// Limit number of copy routines based on available CPU resources.
	cpQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
	defer close(cpQueue)

	// Status channel for receiveing copy return status.
	statusCh := make(chan copyURLs)

	// Go routine to monitor doCopy status and signal traps.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case cpURLs, ok := <-statusCh: // Receive status.
				if !ok { // We are done here. Top level function has returned.
					if globalJSONFlag {
						return
					}
					if globalQuietFlag {
						console.Println(console.Colorize("Copy", progressReader.(*accounter).Finish()))
						return
					}
					progressReader.(*barSend).Finish()
					return
				}
				if cpURLs.Error == nil {
					session.Header.LastCopied = cpURLs.SourceContent.URL.String()
					session.Save()
				} else {
					// Print in new line and adjust to top so that we don't print over the ongoing progress bar
					if !globalQuietFlag && !globalJSONFlag {
						console.Eraseline()
					}
					errorIf(cpURLs.Error.Trace(), fmt.Sprintf("Failed to copy ‘%s’.", cpURLs.SourceContent.URL.String()))
					// all the cases which are handled where session should be saved are contained in the following
					// switch case, we shouldn't be saving sessions for all errors since some errors might need to be
					// reported to user properly.
					//
					// All other critical cases should be handled properly gracefully
					// handle more errors and save the session.
					switch cpURLs.Error.ToGoError().(type) {
					case *net.OpError:
						session.CloseAndDie()
					case net.Error:
						session.CloseAndDie()
					}
				}
			case <-trapCh: // Receive interrupt notification.
				if !globalQuietFlag && !globalJSONFlag {
					console.Eraseline()
				}
				session.CloseAndDie()
			}
		}
	}()

	// Go routine to perform concurrently copying.
	wg.Add(1)
	go func() {
		defer wg.Done()
		copyWg := new(sync.WaitGroup)
		defer close(statusCh)

		for scanner.Scan() {
			var cpURLs copyURLs
			json.Unmarshal([]byte(scanner.Text()), &cpURLs)
			if isCopied(cpURLs.SourceContent.URL.String()) {
				doCopyFake(cpURLs, progressReader)
			} else {
				// Wait for other copy routines to
				// complete. We only have limited CPU
				// and network resources.
				cpQueue <- true
				// Account for each copy routines we start.
				copyWg.Add(1)
				// Do copying in background concurrently.
				go doCopy(cpURLs, progressReader, cpQueue, copyWg, statusCh)
			}
		}
		copyWg.Wait()
	}()
	wg.Wait()
}

// mainCopy is the entry point for cp command.
func mainCopy(ctx *cli.Context) {
	checkCopySyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Copy", color.New(color.FgGreen, color.Bold))

	var e error
	session := newSessionV3()
	session.Header.CommandType = "cp"
	session.Header.RootPath, e = os.Getwd()
	if e != nil {
		session.Delete()
		fatalIf(probe.NewError(e), "Unable to get current working folder.")
	}

	// extract URLs.
	var err *probe.Error
	session.Header.CommandArgs, err = args2URLs(ctx.Args())
	if err != nil {
		session.Delete()
		fatalIf(err.Trace(), "One or more unknown URL types passed.")
	}

	doCopySession(session)
	session.Delete()
}
